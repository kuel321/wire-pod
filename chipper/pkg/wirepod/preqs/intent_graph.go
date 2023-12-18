package processreqs

import (
	//"runtime/debug"

	"github.com/kercre123/wire-pod/chipper/pkg/logger"
	"github.com/kercre123/wire-pod/chipper/pkg/vars"
	"github.com/kercre123/wire-pod/chipper/pkg/vtt"
	sr "github.com/kercre123/wire-pod/chipper/pkg/wirepod/speechrequest"
	ttr "github.com/kercre123/wire-pod/chipper/pkg/wirepod/ttr"
)

func (s *Server) ProcessIntentGraph(req *vtt.IntentGraphRequest) (*vtt.IntentGraphResponse, error) {
	//robotObj, robotIndex, err := getRobot("007077a9")
	//robot := robotObj.Vector
	//ctx := robotObj.Ctx
	var successMatched bool
	//logger.Println(err)
	speechReq := sr.ReqToSpeechRequest(req)
	logger.Println(req)
	logger.Println("line 19 intent_graph.go")

	var transcribedText string
	if !isSti {
		var err error
		transcribedText, err = sttHandler(speechReq)
		if err != nil {
			ttr.IntentPass(req, "intent_system_noaudio", "voice processing error", map[string]string{"error": err.Error()}, true)
			return nil, nil
		}
		successMatched = ttr.ProcessTextAll(req, transcribedText, vars.MatchListList, vars.IntentsList, speechReq.IsOpus)
	} else {
		intent, slots, err := stiHandler(speechReq)
		if err != nil {
			if err.Error() == "inference not understood" {
				logger.Println("Bot " + speechReq.Device + " No intent was matched")
				ttr.IntentPass(req, "intent_system_noaudio", "voice processing error", map[string]string{"error": err.Error()}, true)
				return nil, nil
			}
			logger.Println(err)
			ttr.IntentPass(req, "intent_system_noaudio", "voice processing error", map[string]string{"error": err.Error()}, true)
			return nil, nil
		}
		ttr.ParamCheckerSlotsEnUS(req, intent, slots, speechReq.IsOpus, speechReq.Device)
		return nil, nil
	}
	if !successMatched {
		logger.Println("No intent was matched.")

		apiResponse := openaiRequest(transcribedText)

		//audioFile := "./test.mp3"
		//logger.Println("/home/luke/wire-pod/chipper/pkg/wirepod/preqs/output/test.wav")

		//pkg\wirepod\preqs\output\test.wav

		// robot.Conn.SayText(
		// 	ctx,
		// 	&vectorpb.SayTextRequest{
		// 		DurationScalar: 1,
		// 		UseVectorVoice: true,
		// 		Text:           apiResponse,
		// 	},
		// )

		logger.Println(apiResponse)

		// req.Stream.Send(apiResponse)

		//robots[robotIndex].BcAssumption = false

		//ttr.IntentPass(req, "intent_system_noaudio", transcribedText, map[string]string{"": ""}, false)

		return nil, nil
	}
	logger.Println("Bot " + speechReq.Device + " request served.")
	return nil, nil

}
