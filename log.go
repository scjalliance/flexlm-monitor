package flexlm

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/hpcloud/tail"
)

// LogOptions provide a way to set options
type LogOptions struct {
	Timezone                *time.Location
	StopChan                chan struct{}
	ReportParsingErrors     bool
	ReportUnmatchedLogLines bool
}

// LogItem is a generic log entry
type LogItem struct {
	When        time.Time
	Service     string
	Direction   int
	LicenseName string
	Username    string
	Machine     string
	Error       error
	RawLine     string
}

// Directions
const (
	DirectionUnmatched int = iota - 2
	DirectionDenied
	DirectionIn
	DirectionOut
)

// errors
var (
	ErrUnmatchedLine = errors.New("unmatched log line")
)

var (
	logItemPrefix      = `^\s*([0-9]{1,2}:[0-9]{2}:[0-9]{2})\s+\((adskflex|lmgrd)\)\s+`
	logItemPrefixMatch = regexp.MustCompile(logItemPrefix)
	logItemStarted     = regexp.MustCompile(logItemPrefix + `FlexNet\s+Licensing\s+.*\(([0-9]{1,2}/[0-9]{1,2}/[0-9]{4})\)\s*$`)
	logItemTimestamp   = regexp.MustCompile(logItemPrefix + `TIMESTAMP\s+([0-9]{1,2}/[0-9]{1,2}/[0-9]{4})`)
	logItemInOutDenied = regexp.MustCompile(logItemPrefix + `(IN|OUT|DENIED):\s+"([^"]+)"\s+(\S+)@(\S+)(\s+\((.*)\))?\s*$`)
)

// TailLog tails a log
func TailLog(filename string, opts *LogOptions) (logChan chan LogItem, err error) {
	t, err := tail.TailFile(filename, tail.Config{
		MustExist: true,
		Follow:    true,
		ReOpen:    true,
	})
	if err != nil {
		return nil, err
	}

	if opts.StopChan != nil {
		go func() {
			select {
			case <-t.Tomb.Dying():
				// we're done here
			case <-opts.StopChan:
				// please be done here
				t.Tomb.Kill(nil)
			}
		}()
	}

	if opts == nil {
		opts = &LogOptions{}
	}

	if opts.Timezone == nil {
		opts.Timezone = time.UTC
	}

	logChan = make(chan LogItem)

	var currentTimestamp time.Time

	go func() {
		for line := range t.Lines {
			item := LogItem{
				RawLine: line.Text,
			}

			// STARTED
			if m := logItemStarted.FindAllStringSubmatch(line.Text, 1); len(m) > 0 {
				timeString := m[0][1]
				dateString := m[0][3]
				t, err := time.ParseInLocation("1/2/2006 15:04:05", dateString+" "+timeString, opts.Timezone)
				if err != nil && opts.ReportParsingErrors {
					item.Error = err
					logChan <- item
					continue
				}
				currentTimestamp = t
				continue
			}

			// TIMESTAMP
			if m := logItemTimestamp.FindAllStringSubmatch(line.Text, 1); len(m) > 0 {
				timeString := m[0][1]
				dateString := m[0][3]
				t, err := time.ParseInLocation("1/2/2006 15:04:05", dateString+" "+timeString, opts.Timezone)
				if err != nil && opts.ReportParsingErrors {
					item.Error = err
					logChan <- item
					continue
				}
				currentTimestamp = t
				continue
			}

			// IN|OUT|DENIED
			if m := logItemInOutDenied.FindAllStringSubmatch(line.Text, 1); len(m) > 0 {
				// Time
				timeString := m[0][1]
				cty, ctm, ctd := currentTimestamp.Date()
				t, err := time.ParseInLocation("2006/1/2 15:04:05", fmt.Sprintf("%d/%d/%d %s", cty, ctm, ctd, timeString), opts.Timezone)
				if err != nil && opts.ReportParsingErrors {
					item.Error = err
					logChan <- item
					continue
				}
				if t.Hour() < currentTimestamp.Hour() {
					// compensate for move to new day prior to a new TIMESTAMP log entry
					t = t.Add(time.Hour * 24)
				}
				item.When = t

				// Direction
				direction := m[0][3]
				switch direction {
				case "IN":
					item.Direction = DirectionIn
				case "OUT":
					item.Direction = DirectionOut
				case "DENIED":
					item.Direction = DirectionDenied
				default:
					// this should be impossible as long as the RegExp logItemInOutDenied is in sync with this switch...
					item.Error = fmt.Errorf("Direction [%s] invalid.", direction)
					if opts.ReportParsingErrors {
						logChan <- item
					}
					continue
				}

				// LicenseName
				item.LicenseName = m[0][4]

				// Username
				item.Username = m[0][5]

				// Machine
				item.Machine = m[0][6]

				// Error
				err = errors.New(m[0][8])

				if err != nil {
					item.Error = err
				}

				logChan <- item
				continue
			}

			// we hit a line we don't match
			item.Direction = DirectionUnmatched
			if m := logItemPrefixMatch.FindAllStringSubmatch(line.Text, 1); len(m) > 0 {
				timeString := m[0][1]
				cty, ctm, ctd := currentTimestamp.Date()
				t, err := time.ParseInLocation("2006/1/2 15:04:05", fmt.Sprintf("%d/%d/%d %s", cty, ctm, ctd, timeString), opts.Timezone)
				if err != nil && opts.ReportParsingErrors {
					item.Error = err
					logChan <- item
					continue
				}
				currentTimestamp = t
				item.When = t
			}
			// and then we report the line, if we were told to do so
			if opts.ReportUnmatchedLogLines {
				item.Error = ErrUnmatchedLine
				logChan <- item
			}
		}
		close(logChan)
	}()
	return logChan, nil
}
