package main

import (
	"github.com/argusdusty/sapip"
	"strings"
	"time"
)

var ExampleQueue = new(sapip.Queue)

const ExampleDelay = 1000 * time.Millisecond
const ExampleSimultaneousLimit = 100

type Element struct {
	Name     string
	Data     string
	Priority int
}

var Data = []Element{
	Element{"1", "a", 2},
	Element{"1", "b", 3},
	Element{"2", "a", 2},
	Element{"2", "a", 1},
	Element{"3", "a", 2},
	Element{"3", "a", 3},
	Element{"4", "a", 2},
	Element{"4", "a", 2},
	Element{"5", "a", 2},
	Element{"5", "b", 2},
	Element{"6", "b", 2},
	Element{"6", "a", 2},
}

func init() {
	ExampleCommand := func(name string, data []string) string { return strings.Join(data, " ") + " Done!" }
	go ExampleQueue.InitAndRun(ExampleDelay, ExampleSimultaneousLimit, ExampleCommand)
}

func main() {
	for _, e := range Data {
		sr, i := ExampleQueue.AddElement(e.Name, e.Data, e.Priority)
		print("Insert: Name: ", e.Name, "Data: ", e.Data, "Priority: ", e.Priority, "Index: ", i)
		go func() { print(sr.Read(), "\n") }()
	}
	time.Sleep(13 * time.Second)
	return
}
