package main

import (
	"fmt"
	"time"
)

// Infrastructure
type EventSource interface {
	Setup(func())
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
	for _, v := range w.Provs {
		v.Visit(w)
	}
}

type Visitor interface {
	Visit(i *Walker)
}

func VisitorFactory() func(int, []Visitor) func() {
	a := 0
	return func(i int, provs []Visitor) func() {
		return func() {
			v := &Walker{a, make(map[string]func() interface{}), provs[i:]}
			a += 1
			go v.Walk()
		}
	}
}

// Providers
type Ticker struct {
	howOften time.Duration
}

func (t *Ticker) Setup(visit func()) {
	for {
		time.Sleep(t.howOften)
		visit()
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

// Setup
func main() {
	provs := []Visitor{
		&Ticker{howOften: 300 * time.Millisecond},
		&Waiter{howLong: 700 * time.Millisecond},
		&Printer{}}

	factory := VisitorFactory()
	for n, t := range provs {
		if source, ok := t.(EventSource); ok {
			go source.Setup(factory(n, provs))
		}
	}
	select {}
}
