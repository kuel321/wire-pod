package processreqs

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/fforchino/vector-go-sdk/pkg/vector"
	"github.com/fforchino/vector-go-sdk/pkg/vectorpb"
	"github.com/kercre123/wire-pod/chipper/pkg/logger"
	"github.com/kercre123/wire-pod/chipper/pkg/vars"
)

var BotsToInterrupt struct {
	ESNs []string
}

func ShouldBeInterrupted(esn string) bool {
	for _, sn := range BotsToInterrupt.ESNs {
		if esn == sn {
			RemoveFromInterrupt(esn)
			return true
		}
	}
	return false
}

func Interrupt(esn string) {
	BotsToInterrupt.ESNs = append(BotsToInterrupt.ESNs, esn)
}

func RemoveFromInterrupt(esn string) {
	var newList []string
	for _, bot := range BotsToInterrupt.ESNs {
		if bot != esn {
			newList = append(newList, bot)
		}
	}
	BotsToInterrupt.ESNs = newList
}

func KGSim(esn string, textToSay string) error {
	logger.Printl(textToSay)
	return nil
}
