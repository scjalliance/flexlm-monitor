package flexlm

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/hpcloud/tail"
)

// LogItem is a generic log entry
type LogItem struct {
	When        time.Time
	Service     string
	Direction   int
	LicenseName string
	Username    string
	Machine     string
	Error       string
}

// Directions
const (
	DirectionDenied int = iota - 1
	DirectionIn
	DirectionOut
)

var (
	logItemPrefix      = `^\s*([0-9]{1,2}:[0-9]{2}:[0-9]{2})\s+\((adskflex|lmgrd)\)\s+`
	logItemStarted     = regexp.MustCompile(logItemPrefix + `FlexNet\s+Licensing\s+.*\(([0-9]{1,2}/[0-9]{1,2}/[0-9]{4})\)\s*$`)
	logItemTimestamp   = regexp.MustCompile(logItemPrefix + `TIMESTAMP\s+([0-9]{1,2}/[0-9]{1,2}/[0-9]{4})`)
	logItemInOutDenied = regexp.MustCompile(logItemPrefix + `(IN|OUT|DENIED):\s+"([^"]+)"\s+(\S+)@(\S+)(\s+\((.*)\))?\s*$`)
)

// TailLog tails a log
func TailLog(filename string, timezone *time.Location) error {
	t, err := tail.TailFile(filename, tail.Config{Follow: true})
	if err != nil {
		return err
	}

	var currentTimestamp time.Time

	for line := range t.Lines {
		// fmt.Println(line.Text)

		// STARTED
		if m := logItemStarted.FindAllStringSubmatch(line.Text, 1); len(m) > 0 {
			timeString := m[0][1]
			dateString := m[0][3]
			t, err := time.ParseInLocation("1/2/2006 15:04:05", dateString+" "+timeString, timezone)
			if err != nil {
				return err
			}
			currentTimestamp = t
			// fmt.Printf("STARTED: %+v\n", currentTimestamp)
		}

		// TIMESTAMP
		if m := logItemTimestamp.FindAllStringSubmatch(line.Text, 1); len(m) > 0 {
			timeString := m[0][1]
			dateString := m[0][3]
			t, err := time.ParseInLocation("1/2/2006 15:04:05", dateString+" "+timeString, timezone)
			if err != nil {
				return err
			}
			currentTimestamp = t
			// fmt.Printf("TIMESTAMP: %+v\n", currentTimestamp)
		}

		// IN|OUT
		if m := logItemInOutDenied.FindAllStringSubmatch(line.Text, 1); len(m) > 0 {
			item := LogItem{}

			// Time
			timeString := m[0][1]
			cty, ctm, ctd := currentTimestamp.Date()
			t, err := time.ParseInLocation("2006/1/2 15:04:05", fmt.Sprintf("%d/%d/%d %s", cty, ctm, ctd, timeString), timezone)
			if err != nil {
				return err
			}
			if t.Hour() < currentTimestamp.Hour() {
				// compensate for move to new day prior to a new TIMESTAMP log entry
				t = t.Add(time.Hour * 24)
			}
			item.When = t

			// Direction
			direction := m[0][3]
			if direction == "IN" {
				item.Direction = DirectionIn
			} else if direction == "OUT" {
				item.Direction = DirectionOut
			} else if direction == "DENIED" {
				item.Direction = DirectionDenied
			} else {
				log.Fatalf("Direction [%s] invalid.\n", direction) // FIXME: do we really blow up here?
			}

			// LicenseName
			item.LicenseName = m[0][4]

			// Username
			item.Username = m[0][5]

			// Machine
			item.Machine = m[0][6]

			// Error
			item.Error = m[0][8]

			if item.Error != "" {
				fmt.Printf("ERROR %+v\n", item)
			} else {
				fmt.Printf("INOUT %+v\n", item)
			}
		}
	}
	return nil
}
