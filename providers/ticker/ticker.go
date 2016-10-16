package ticker

import (
	"encoding/json"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/Kozical/taskengine/core/runner"
)

var tickerLock sync.Mutex
var tickers []*time.Ticker

/*

type Provider interface {
	Execute(*Job) error
}

type EventProvider interface {
	Register(func() *Job)
}

*/

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
	return new(TickerProvider)
}

func (tp *TickerProvider) Execute(j *runner.Job) error {
	return nil
}

func (tp *TickerProvider) Register(fn func() *runner.Job) {
	var err error
	job := fn()

	var task *runner.Task
	for _, t := range job.Tasks {
		if t.Provider == tp {
			task = &t
			break
		}
	}

	err = json.Unmarshal(task.Properties, &tp.Settings)
	if err != nil {
		log.Printf("Failed to unmarshal TickerProvider properties -> %v\n", err)
		return
	}
	if len(tp.Settings.Interval) > 0 {
		tp.interval, err = strconv.Atoi(tp.Settings.Interval)
		if err != nil {
			log.Printf("Failed to convert Interval to integer -> %v\n", err)
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
		log.Println("Interval must be set on TickerProvider")
		return
	}

	ticker := time.NewTicker(time.Duration(tp.interval * tp.period))

	func(C <-chan time.Time, j *runner.Job) {
		for {
			select {
			case _, ok := <-C:
				if !ok {
					return
				}
				j.Run()
			}
		}
	}(ticker.C, job)
}
