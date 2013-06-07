package main

import (
	"github.com/argusdusty/sapip"
	"strings"
	"time"
)

type SElement struct {
	Name     string
	Data     string
	Priority int
}

var Data = []SElement{
	SElement{"1", "a", 2},
	SElement{"1", "b", 3},
	SElement{"2", "a", 2},
	SElement{"2", "a", 1},
	SElement{"3", "a", 2},
	SElement{"3", "a", 3},
	SElement{"4", "a", 1},
	SElement{"4", "a", 1},
	SElement{"5", "a", 2},
	SElement{"5", "b", 2},
	SElement{"6", "b", 1},
	SElement{"6", "a", 1},
	SElement{"7", "b", 1},
	SElement{"7", "a", 0},
}

var ExampleQueue = new(Queue)

const ExampleDelay = 1000 * time.Millisecond
const ExampleSimultaneousLimit = 100

func init() {
	ExampleCommand := func(name string, data []string) string { return name + " " + strings.Join(data, " ") + " Done!" }
	ExampleQueue.Init(ExampleSimultaneousLimit, ExampleCommand)
	go ExampleQueue.Run(ExampleDelay)
}

func main() {
	for _, e := range Data {
		sr, i := ExampleQueue.AddElement(e.Name, e.Data, e.Priority)
		print("Insert: Name: ", e.Name, " Data: ", e.Data, " Priority: ", e.Priority, " Index: ", i, "\n")
		go func() { print(sr.Read(), "\n") }()
	}
	time.Sleep(10 * time.Second)
	return
}
