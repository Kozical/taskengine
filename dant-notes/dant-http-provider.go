package main

import (
	"fmt"
	"net/http"
	"time"
)

// Infrastructure
type EventSource interface {
	Setup(func() *Walker)
}

type Walker struct {
	ID int
	// Allow for lazy-loading of any kind of data
	Data  map[string]func() interface{}
	Provs []Visitor
}

func (w *Walker) Have(what string, val func() interface{}) {
	w.Data[what] = val
}

func (w *Walker) Walk() {
	go func() {
		for _, v := range w.Provs {
			v.Visit(w)
		}
	}()
}

type Visitor interface {
	Visit(i *Walker)
}

func VisitorFactory() func(int, []Visitor) func() *Walker {
	a := 0
	return func(i int, provs []Visitor) func() *Walker {
		return func() *Walker {
			v := &Walker{a, make(map[string]func() interface{}), provs[i:]}
			a += 1
			return v
		}
	}
}

// Providers
type RequestHandler struct {
	Path   string
	Walker func() *Walker
}

func (h *RequestHandler) Setup(walker func() *Walker) {
	h.Walker = walker
	http.Handle(h.Path, h)
}

func (h *RequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	walker := h.Walker()
	walker.Have("W", func() interface{} { return w })
	walker.Have("R", func() interface{} { return r })
	fin := make(chan struct{}, 0)
	walker.Have("FINISH", func() interface{} {
		fin <- struct{}{}
		return nil
	})
	fmt.Println(walker.ID, "REQUEST RECIEVIED")
	walker.Walk()
	<-fin
}

func (h *RequestHandler) Visit(i *Walker) {
}

type Ticker struct {
	howOften time.Duration
}

func (t *Ticker) Setup(walker func() *Walker) {
	for {
		time.Sleep(t.howOften)
		w := walker()
		w.Walk()
	}
}

func (t *Ticker) Visit(i *Walker) {
	n := time.Now()
	fmt.Println(i.ID, "Ticker ticked at", n)
	i.Have("time", func() interface{} { return n })
}

type Waiter struct {
	howLong time.Duration
}

func (w *Waiter) Visit(i *Walker) {
	fmt.Println(i.ID, "Waiter waiting", w.howLong)
	time.Sleep(w.howLong)
	fmt.Println(i.ID, "Waiter finished")
}

type Printer struct {
}

func (p *Printer) Visit(i *Walker) {
	t := i.Data["time"]()
	fmt.Println(i.ID, "Time was", t.(time.Time))
}

type PrintToHttp struct {
}

func (p *PrintToHttp) Visit(i *Walker) {
	i.Data["W"]().(http.ResponseWriter).Write([]byte("HELLO WORLD"))
	i.Data["FINISH"]()
}

// Setup
func main() {
	provs := []Visitor{
		&RequestHandler{Path: "/"},
		&PrintToHttp{},
		&Ticker{howOften: 300 * time.Millisecond},
		&Waiter{howLong: 400 * time.Millisecond},
		&Printer{}}

	factory := VisitorFactory()
	for n, t := range provs {
		if j, source := t.(EventSource); source {
			go j.Setup(factory(n, provs))
		}
	}
	http.ListenAndServe(":8080", nil)
}
