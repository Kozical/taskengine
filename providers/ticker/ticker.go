package ticker

import (
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/Kozical/taskengine/core/runner"
)

var tickerLock sync.Mutex
var tickers []*time.Ticker

/*
type Provider interface {
	Execute(*Job) (StateObject, error)
	Register(*Job, json.RawMessage) error
	New() Provider
	Name() string
	Cleanup()
}
*/

// TickerState: implements core.StateObject interface
type TickerState struct {
	Time time.Time
}

func (t TickerState) GetProperty(name string) interface{} {
	return name
}

// TickerProvider: implements core.Provider interface
type TickerProvider struct {
	Settings struct {
		Interval string `json:"Interval"`
		Period   string `json:"Period"`
	}
	interval int
	period   int
}

func NewTickerProvider() *TickerProvider {
	return &TickerProvider{}
}

func (tp *TickerProvider) Cleanup() {
	tickerLock.Lock()
	defer tickerLock.Unlock()

	for i := len(tickers); i >= 0; i-- {
		if tickers[i] != nil {
			tickers[i].Stop()
			tickers[i] = nil
		}
	}
}

func (tp *TickerProvider) Name() string {
	return "ticker"
}

func (tp *TickerProvider) New() runner.Provider {
	return &TickerProvider{}
}

func (tp *TickerProvider) Register(j *runner.Job, raw json.RawMessage) (err error) {
	err = json.Unmarshal(raw, &tp.Settings)
	if err != nil {
		return
	}
	if len(tp.Settings.Interval) > 0 {
		tp.interval, err = strconv.Atoi(tp.Settings.Interval)
		if err != nil {
			return
		}
	}
	switch tp.Settings.Period {
	case "Second":
		tp.period = int(time.Second)
	case "Millisecond":
		tp.period = int(time.Millisecond)
	case "Minute":
		tp.period = int(time.Minute)
	case "Hour":
		tp.period = int(time.Hour)
	case "Day":
		tp.period = int(24 * time.Hour)
	default:
		tp.period = int(time.Second)
	}
	if tp.interval == 0 {
		err = errors.New("Interval must be set on ticker")
		return
	}
	return
}

//Execute(*Job) (StateObject, error)

func (tp *TickerProvider) Execute(j *runner.Job) (state runner.StateObject, err error) {
	ticker := time.NewTicker(time.Duration(tp.interval * tp.period))

	tickerLock.Lock()
	tickers = append(tickers, ticker)
	tickerLock.Unlock()

	go func(C <-chan time.Time, j *runner.Job) {
		for {
			select {
			case t, ok := <-C:
				if !ok {
					return
				}
				state = TickerState{
					Time: t,
				}
				j.Run(tp)
			}
		}
	}(ticker.C, j)

	return
}
