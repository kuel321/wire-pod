package processreqs

import (
	"bytes"
	"encoding/json"

	"io"
	"net/http"

	//"os"

	"strings"

	pb "github.com/digital-dream-labs/api/go/chipperpb"

	//"github.com/hajimehoshi/go-mp3"
	"github.com/kercre123/wire-pod/chipper/pkg/logger"
	"github.com/kercre123/wire-pod/chipper/pkg/vars"
	"github.com/kercre123/wire-pod/chipper/pkg/vtt"
	sr "github.com/kercre123/wire-pod/chipper/pkg/wirepod/speechrequest"
	"github.com/pkg/errors"
	"github.com/soundhound/houndify-sdk-go"
)

var HKGclient houndify.Client
var HoundEnable bool = true

func ParseSpokenResponse(serverResponseJSON string) (string, error) {
	result := make(map[string]interface{})
	err := json.Unmarshal([]byte(serverResponseJSON), &result)
	if err != nil {
		logger.Println(err.Error())
		return "", errors.New("failed to decode json")
	}
	if !strings.EqualFold(result["Status"].(string), "OK") {
		return "", errors.New(result["ErrorMessage"].(string))
	}
	if result["NumToReturn"].(float64) < 1 {
		return "", errors.New("no results to return")
	}
	return result["AllResults"].([]interface{})[0].(map[string]interface{})["SpokenResponseLong"].(string), nil
}

func InitKnowledge() {
	if vars.APIConfig.Knowledge.Enable && vars.APIConfig.Knowledge.Provider == "houndify" {
		if vars.APIConfig.Knowledge.ID == "" || vars.APIConfig.Knowledge.Key == "" {
			vars.APIConfig.Knowledge.Enable = false
			logger.Println("Houndify Client Key or ID was empty, not initializing kg client")
		} else {
			HKGclient = houndify.Client{
				ClientID:  vars.APIConfig.Knowledge.ID,
				ClientKey: vars.APIConfig.Knowledge.Key,
			}
			HKGclient.EnableConversationState()
			logger.Println("Initialized Houndify client")
		}
	}
}

var NoResult string = "NoResultCommand"
var NoResultSpoken string

func houndifyKG(req sr.SpeechRequest) string {
	var apiResponse string
	if vars.APIConfig.Knowledge.Enable && vars.APIConfig.Knowledge.Provider == "houndify" {
		logger.Println("Sending request to Houndify...")
		serverResponse := StreamAudioToHoundify(req, HKGclient)
		apiResponse, _ = ParseSpokenResponse(serverResponse)
		logger.Println("Houndify response: " + apiResponse)
	} else {
		apiResponse = "Houndify is not enabled."
		logger.Println("Houndify is not enabled.")
	}
	return apiResponse
}

func togetherRequest(transcribedText string) string {
	sendString := "You are an assistant named Dave. You  Your responses will be sent to a generated voice so make it sound realistic, add uhms and uhs, and add inflection in the text response, as if it was being said by a real person. My questions will be sent with STT so It may not be punctuated or spelled correctly. You have a fun personality. Here is the question: " + "\\" + "\"" + transcribedText + "\\" + "\"" + " , Answer: "
	url := "https://api.together.xyz/inference"
	model := vars.APIConfig.Knowledge.Model
	formData := `{
"model": "` + model + `",
"prompt": "` + sendString + `",
"temperature": 0.7,
"max_tokens": 256,
"top_p": 1
}`
	logger.Println("Making request to Together API...")
	logger.Println("Model is " + model)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte(formData)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+vars.APIConfig.Knowledge.Key)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "There was an error making the request to Together API"
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var togetherResponse map[string]any
	err = json.Unmarshal(body, &togetherResponse)
	if err != nil {
		return "Together API returned no response."
	}
	output := togetherResponse["output"].(map[string]any)
	choice := output["choices"].([]any)
	for _, val := range choice {
		x := val.(map[string]any)
		textResponse := x["text"].(string)
		apiResponse := strings.TrimSuffix(textResponse, "</s>")
		logger.Println("Together response: " + apiResponse)
		return apiResponse
	}
	// In case text is not present in result from API, return a string saying answer was not found
	return "Answer was not found"
}

func textToSpeechOpenAi(openAIResponse string) error {

	url := "http://escapepod.local:8125/speechcreate"
	formData := `{
		"model": "gpt-3.5-turbo-instruct",
	
		"temperature": 0.7,
		"max_tokens": 256,
		"top_p": 1,
		"frequency_penalty": 0.2,
		"presence_penalty": 0
		}`
	req, _ := http.NewRequest("GET", url, bytes.NewBuffer([]byte(formData)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("text", openAIResponse)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Println(err)
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	testingSDK := PlaySound("/home/luke/tts-api/speech.mp3")
	logger.Println(testingSDK)
	err = json.Unmarshal(body, &openAIResponse)
	if err != nil {
		logger.Println(err)
		//logger.Println(body)
		return err
	}

	return nil
}

func openaiRequest(transcribedText string) string {

	sendString := "You are a robot assistant with a fun personality, but you're also super intelligent. You live in my office, on one of my shelves. Sometimes when I ask a question, say that it's hard to think all the way up there. If I say anything with the word thimble, say that you hate thimbles. When typing out the responses, do not add new paragraphs, make it all one line. I am going to ask a question here and you respond. Here's the question: " + "\\" + "\"" + transcribedText + "\\" + "\"" + " , Answer: "
	logger.Println("Making request to OpenAI...4:49 test")
	url := "https://api.openai.com/v1/completions"
	formData := `{
		"model": "gpt-3.5-turbo-instruct",
		"prompt": "` + sendString + `",
		"temperature": 1,
		"max_tokens": 256,
		"top_p": 1,
		"frequency_penalty": 0.2,
		"presence_penalty": 0
		}`
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte(formData)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+vars.APIConfig.Knowledge.Key)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Println(err)
		return "There was an error making the request to OpenAI."
	}
	logger.Println("OpenAI Header " + sendString)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	type openAIStruct struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int    `json:"created"`
		Model   string `json:"model"`
		Choices []struct {
			Text         string      `json:"text"`
			Index        int         `json:"index"`
			Logprobs     interface{} `json:"logprobs"`
			FinishReason string      `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	var openAIResponse openAIStruct
	err = json.Unmarshal(body, &openAIResponse)
	if err != nil || len(openAIResponse.Choices) == 0 {
		logger.Println("OpenAI returned no response.")
		return "OpenAI returned no response."
	}
	apiResponse := strings.TrimSpace(openAIResponse.Choices[0].Text)
	textToSpeechOpenAi(apiResponse)
	logger.Println("trying to run text to speech function api call test")

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

// Takes a SpeechRequest, figures out knowledgegraph provider, makes request, returns API response
func KgRequest(speechReq sr.SpeechRequest) string {
	if vars.APIConfig.Knowledge.Enable {
		if vars.APIConfig.Knowledge.Provider == "houndify" {
			return houndifyKG(speechReq)
		} else if vars.APIConfig.Knowledge.Provider == "openai" {
			return openaiKG(speechReq)
		} else if vars.APIConfig.Knowledge.Provider == "together" {
			return togetherKG(speechReq)
		}
	}
	return "Knowledge graph is not enabled. This can be enabled in the web interface."
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
	logger.Println("(KG) Bot " + speechReq.Device + " request served.")
	if err := req.Stream.Send(&kg); err != nil {
		return nil, err
	}
	return nil, nil

}
