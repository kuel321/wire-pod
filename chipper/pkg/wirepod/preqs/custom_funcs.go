package processreqs

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/fforchino/vector-go-sdk/pkg/vectorpb"

	"github.com/kercre123/wire-pod/chipper/pkg/logger"
)

var SYSTEMSOUND_WIN = "audio/win.pcm"

const VOLUME_LEVEL_MAXIMUM = 5
const VOLUME_LEVEL_MINIMUM = 1

var audioStreamClient vectorpb.ExternalInterface_AudioFeedClient
var audioStreamEnable bool = false

func EnableAudioStream() {
	robotObj, robotIndex, err := getRobot("007077a9")
	robot := robotObj.Vector
	ctx := robotObj.Ctx
	audioStreamClient, _ = robot.Conn.AudioFeed(ctx, &vectorpb.AudioFeedRequest{})
	audioStreamEnable = true
	logger.Println(robotIndex, err)
}

func DisableAudioStream() {
	audioStreamEnable = false
	audioStreamClient = nil
}

func ProcessAudioStream() {
	if audioStreamEnable {
		response, _ := audioStreamClient.Recv()
		audioSample := response.SignalPower
		println(string(audioSample))
	}
}

// Returns values in the range 1-5

// Returns values in the range 0-100

// Plays one of the SYSTEMSOUND_... files

func PlaySound(filename string) string {

	logger.Println(filename)
	robotObj, robotIndex, err := getRobot("007077a9")
	robot := robotObj.Vector
	ctx := robotObj.Ctx
	logger.Println(robotIndex, err)
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		println("File not found!")
		return "failure"
	}

	var pcmFile []byte
	tmpFileName := "./output/test.wav"
	//fmt.Println("FFMPEG output: " + string(conOutput))
	pcmFile, _ = os.ReadFile(tmpFileName)

	var audioChunks [][]byte
	for len(pcmFile) >= 1024 {
		audioChunks = append(audioChunks, pcmFile[:1024])
		pcmFile = pcmFile[1024:]
	}
	var audioClient vectorpb.ExternalInterface_ExternalAudioStreamPlaybackClient
	audioClient, _ = robot.Conn.ExternalAudioStreamPlayback(
		ctx,
	)
	audioClient.SendMsg(&vectorpb.ExternalAudioStreamRequest{
		AudioRequestType: &vectorpb.ExternalAudioStreamRequest_AudioStreamPrepare{
			AudioStreamPrepare: &vectorpb.ExternalAudioStreamPrepare{
				AudioFrameRate: 16000,
				AudioVolume:    100,
			},
		},
	})
	//fmt.Println(len(audioChunks))
	for _, chunk := range audioChunks {
		audioClient.SendMsg(&vectorpb.ExternalAudioStreamRequest{
			AudioRequestType: &vectorpb.ExternalAudioStreamRequest_AudioStreamChunk{
				AudioStreamChunk: &vectorpb.ExternalAudioStreamChunk{
					AudioChunkSizeBytes: 1024,
					AudioChunkSamples:   chunk,
				},
			},
		})
		time.Sleep(time.Millisecond * 30)
	}
	audioClient.SendMsg(&vectorpb.ExternalAudioStreamRequest{
		AudioRequestType: &vectorpb.ExternalAudioStreamRequest_AudioStreamComplete{
			AudioStreamComplete: &vectorpb.ExternalAudioStreamComplete{},
		},
	})
	os.Remove(tmpFileName)

	return "success"
}
func assumeBehaviorControl(robot Robot, robotIndex int, priority string) {
	var controlRequest *vectorpb.BehaviorControlRequest
	if priority == "high" {
		controlRequest = &vectorpb.BehaviorControlRequest{
			RequestType: &vectorpb.BehaviorControlRequest_ControlRequest{
				ControlRequest: &vectorpb.ControlRequest{
					Priority: vectorpb.ControlRequest_OVERRIDE_BEHAVIORS,
				},
			},
		}
	} else {
		controlRequest = &vectorpb.BehaviorControlRequest{
			RequestType: &vectorpb.BehaviorControlRequest_ControlRequest{
				ControlRequest: &vectorpb.ControlRequest{
					Priority: vectorpb.ControlRequest_DEFAULT,
				},
			},
		}
	}
	go func() {
		start := make(chan bool)
		stop := make(chan bool)
		robots[robotIndex].BcAssumption = true
		go func() {
			// * begin - modified from official vector-go-sdk
			r, err := robot.Vector.Conn.BehaviorControl(
				robot.Ctx,
			)
			if err != nil {
				log.Println(err)
				return
			}

			if err := r.Send(controlRequest); err != nil {
				log.Println(err)
				return
			}

			for {
				ctrlresp, err := r.Recv()
				if err != nil {
					log.Println(err)
					return
				}
				if ctrlresp.GetControlGrantedResponse() != nil {
					start <- true
					break
				}
			}

			for {
				select {
				case <-stop:
					if err := r.Send(
						&vectorpb.BehaviorControlRequest{
							RequestType: &vectorpb.BehaviorControlRequest_ControlRelease{
								ControlRelease: &vectorpb.ControlRelease{},
							},
						},
					); err != nil {
						log.Println(err)
						return
					}
					return
				default:
					continue
				}
			}

		}()
		for range start {
			for {
				if robots[robotIndex].BcAssumption {
					time.Sleep(time.Millisecond * 10000)
				} else {
					break
				}
			}
			stop <- true
		}
	}()
}
