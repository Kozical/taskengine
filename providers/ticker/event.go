package ticker

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/Kozical/taskengine/job"
)

// TickerState: implements job.StateObject interface
type TickerState struct {
	Time time.Time
}

func (t TickerState) GetProperty(name string) string {
	return name
}

// TickerEventProvider: implements job.EventProvider interface
type TickerEventProvider struct {
	Settings struct {
		Interval string `json:"Interval"`
		Period   string `json:"Period"`
	}
	Tickers []*time.Ticker
}

func NewTickerEventProvider() *TickerEventProvider {
	return &TickerEventProvider{}
}

func (tep *TickerEventProvider) Cleanup() {
	for _, t := range tep.Tickers {
		t.Stop()
	}
}

func (tep *TickerEventProvider) Name() string {
	return "ticker_event"
}
func (tep *TickerEventProvider) Event(j *job.Job, raw json.RawMessage, dispatch job.DispatchFunc) (err error) {
	err = json.Unmarshal(raw, &tep.Settings)
	if err != nil {
		return
	}

	var period, interval int
	if len(tep.Settings.Interval) > 0 {
		interval, err = strconv.Atoi(tep.Settings.Interval)
		if err != nil {
			return
		}
	}
	switch tep.Settings.Period {
	case "Second":
		period = int(time.Second)
	case "Millisecond":
		period = int(time.Millisecond)
	case "Minute":
		period = int(time.Minute)
	case "Hour":
		period = int(time.Hour)
	case "Day":
		period = int(24 * time.Hour)
	default:
		period = int(time.Second)
	}

	if interval == 0 {
		err = errors.New("Interval must be set on ticker_event")
		return
	}

	ticker := time.NewTicker(time.Duration(interval * period))

	tep.Tickers = append(tep.Tickers, ticker)

	go func(C <-chan time.Time, j *job.Job) {
		for {
			select {
			case t, ok := <-C:
				if !ok {
					return
				}
				j.Run(TickerState{
					Time: t,
				})
			}
		}
	}(ticker.C, j)

	return nil
}
