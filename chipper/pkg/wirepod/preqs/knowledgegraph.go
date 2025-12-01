package processreqs

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	pb "github.com/digital-dream-labs/api/go/chipperpb"

	"github.com/kercre123/wire-pod/chipper/pkg/logger"
	"github.com/kercre123/wire-pod/chipper/pkg/vars"
	"github.com/kercre123/wire-pod/chipper/pkg/vtt"
	sr "github.com/kercre123/wire-pod/chipper/pkg/wirepod/speechrequest"

	"github.com/pkg/errors"
	"github.com/soundhound/houndify-sdk-go"
)

var HKGclient houndify.Client
var HoundEnable = true

// SAFELY parse Houndify JSON
func ParseSpokenResponse(serverResponseJSON string) (string, error) {
	var result map[string]any

	if err := json.Unmarshal([]byte(serverResponseJSON), &result); err != nil {
		logger.Println(err.Error())
		return "", errors.New("failed to decode json")
	}

	// Safe-assert all needed fields
	status, _ := result["Status"].(string)
	if !strings.EqualFold(status, "OK") {
		errMsg, _ := result["ErrorMessage"].(string)
		if errMsg == "" {
			errMsg = "Unknown error from server"
		}
		return "", errors.New(errMsg)
	}

	num, _ := result["NumToReturn"].(float64)
	if num < 1 {
		return "", errors.New("no results to return")
	}

	all, ok := result["AllResults"].([]any)
	if !ok || len(all) == 0 {
		return "", errors.New("invalid results")
	}

	first, ok := all[0].(map[string]any)
	if !ok {
		return "", errors.New("invalid result structure")
	}

	text, _ := first["SpokenResponseLong"].(string)
	if text == "" {
		return "", errors.New("response missing text")
	}

	return text, nil
}

func InitKnowledge() {
	if vars.APIConfig.Knowledge.Enable && vars.APIConfig.Knowledge.Provider == "houndify" {
		if vars.APIConfig.Knowledge.ID == "" || vars.APIConfig.Knowledge.Key == "" {
			vars.APIConfig.Knowledge.Enable = false
			logger.Println("Houndify Client Key or ID was empty, not initializing client")
			return
		}

		HKGclient = houndify.Client{
			ClientID:  vars.APIConfig.Knowledge.ID,
			ClientKey: vars.APIConfig.Knowledge.Key,
		}

		HKGclient.EnableConversationState()
		logger.Println("Initialized Houndify client")
	}
}

var (
	NoResult       = "NoResultCommand"
	NoResultSpoken string
)

func houndifyKG(req sr.SpeechRequest) string {
	if !vars.APIConfig.Knowledge.Enable || vars.APIConfig.Knowledge.Provider != "houndify" {
		logger.Println("Houndify is not enabled.")
		return "Houndify is not enabled."
	}

	logger.Println("Sending request to Houndify...")
	serverResponse := StreamAudioToHoundify(req, HKGclient)

	apiResponse, err := ParseSpokenResponse(serverResponse)
	if err != nil {
		logger.Println("Houndify error:", err.Error())
		return "There was an error."
	}

	logger.Println("Houndify response:", apiResponse)
	return apiResponse
}

func togetherRequest(transcribedText string) string {
	sendString := `You are an assistant named Dave. Your responses will be sent to a generated voice so make it sound realistic, add uhms and uhs, and inflection. My questions will be noisy STT. Here is the question: "` + transcribedText + `". Answer:` 

	url := "https://api.together.xyz/inference"
	model := vars.APIConfig.Knowledge.Model

	formData := map[string]any{
		"model":     model,
		"prompt":    sendString,
		"temperature": 0.7,
		"max_tokens": 256,
		"top_p":     1,
	}

	payload, _ := json.Marshal(formData)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+vars.APIConfig.Knowledge.Key)

	logger.Println("Making request to Together API with model:", model)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "Error contacting Together API"
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var togetherResponse map[string]any
	if err := json.Unmarshal(body, &togetherResponse); err != nil {
		return "Together API returned no response."
	}

	output, ok := togetherResponse["output"].(map[string]any)
	if !ok {
		return "Answer not found"
	}

	choices, ok := output["choices"].([]any)
	if !ok || len(choices) == 0 {
		return "Answer not found"
	}

	x, _ := choices[0].(map[string]any)
	textResponse, _ := x["text"].(string)
	apiResponse := strings.TrimSuffix(textResponse, "</s>")

	logger.Println("Together response:", apiResponse)
	return apiResponse
}

func textToSpeechOpenAi(text string) error {
	url := "http://escapepod.local:8125/speechcreate"

	form := map[string]any{
		"model":             "gpt-3.5-turbo-instruct",
		"temperature":       0.7,
		"max_tokens":        256,
		"top_p":             1,
		"frequency_penalty": 0.2,
		"presence_penalty":  0,
	}

	payload, _ := json.Marshal(form)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("text", text)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Println(err)
		return err
	}
	defer resp.Body.Close()

	_, _ = io.ReadAll(resp.Body)

	PlaySound("/home/luke/tts-api/speech.mp3")
	return nil
}

func openaiRequest(transcribedText string) string {
	sendString := `You are an assistant named Vector. You provide helpful answers but not too lengthy since you'll be a voice assistant. Do not use new paragraphs. Here's the question: "` + transcribedText + `". Answer:` 

	url := "https://api.openai.com/v1/completions"

	formData := map[string]any{
		"model":             "gpt-3.5-turbo-instruct",
		"prompt":            sendString,
		"temperature":       1,
		"max_tokens":        256,
		"top_p":             1,
		"frequency_penalty": 0.2,
		"presence_penalty":  0,
	}

	payload, _ := json.Marshal(formData)
	logger.Println("Making request to OpenAI with model:", formData["model"])
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+vars.APIConfig.Knowledge.Key)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Println(err)
		return "Error contacting OpenAI."
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	type openAIStruct struct {
		Choices []struct {
			Text string `json:"text"`
		} `json:"choices"`
	}

	var parsed openAIStruct
	if json.Unmarshal(body, &parsed) != nil || len(parsed.Choices) == 0 {
		return "OpenAI returned no response."
	}

	apiResponse := strings.TrimSpace(parsed.Choices[0].Text)
	_ = textToSpeechOpenAi(apiResponse)

	return apiResponse
}

func openaiKG(speechReq sr.SpeechRequest) string {
	transcribedText, err := sttHandler(speechReq)
	if err != nil {
		return "There was an error."
	}
	return openaiRequest(transcribedText)
}

func togetherKG(speechReq sr.SpeechRequest) string {
	transcribedText, err := sttHandler(speechReq)
	if err != nil {
		return "There was an error."
	}
	return togetherRequest(transcribedText)
}

func KgRequest(speechReq sr.SpeechRequest) string {
	if !vars.APIConfig.Knowledge.Enable {
		return "Knowledge graph is not enabled. Enable it in the web interface."
	}

	switch vars.APIConfig.Knowledge.Provider {
	case "houndify":
		return houndifyKG(speechReq)
	case "openai":
		return openaiKG(speechReq)
	case "together":
		return togetherKG(speechReq)
	}

	return "Unknown provider."
}

func (s *Server) ProcessKnowledgeGraph(req *vtt.KnowledgeGraphRequest) (*vtt.KnowledgeGraphResponse, error) {
	InitKnowledge()

	speechReq := sr.ReqToSpeechRequest(req)
	apiResponse := KgRequest(speechReq)

	kg := pb.KnowledgeGraphResponse{
		Session:     req.Session,
		DeviceId:    req.Device,
		CommandType: NoResult,
		SpokenText:  apiResponse,
	}

	logger.Println("(KG) Bot", speechReq.Device, "request served.")

	if err := req.Stream.Send(&kg); err != nil {
		return nil, err
	}

	return nil, nil
}
