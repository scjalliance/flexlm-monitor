package main

import (
	"fmt"
	"log"

	flexlm "github.com/scjalliance/flexlm-monitor"
)

func main() {
	// stopChan := make(chan struct{})
	logChan, err := flexlm.TailLog("test.log", &flexlm.LogOptions{
		// StopChan: stopChan,
		ReportUnmatchedLogLines: false,
		ReportParsingErrors:     false,
	})
	if err != nil {
		log.Fatal(err)
	}

	// maxI := 10
	// i := 0
	for {
		item, ok := <-logChan
		if !ok {
			log.Fatal("LOGCHAN is closed")
		}
		// i++
		// if i == maxI {
		// 	fmt.Println("CLOSING THE STOPCHAN")
		// 	close(stopChan)
		// }
		var direction string
		if item.Direction == flexlm.DirectionOut {
			direction = "Check-out"
		} else if item.Direction == flexlm.DirectionIn {
			direction = "Check-in"
		} else if item.Direction == flexlm.DirectionDenied {
			direction = "Denied check-out"
		} else {
			fmt.Printf("%+v\n", item)
			continue
		}
		fmt.Printf("%s of %s for %s on %s\n", direction, item.LicenseName, item.Username, item.Machine)
	}
}
