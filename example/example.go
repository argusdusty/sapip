package main

import (
	"github.com/argusdusty/sapip"
	"time"
)

var ExampleQueue = new(sapip.Queue)
const ExampleDelay = 100*time.Millisecond
const ExampleSimultaneousLimit = 100

func init() {
	ExampleQueue.Init(ExampleDelay, ExampleSimultaneousLimit)
	ExampleQueue.Run()
}

func main() {
	ExampleCommand := func(input string) string { return input + " Done!" }
	for i := 10; i > 0; i-- {
		r, _ := ExampleQueue.AddElement("Testing: " + string(byte(i + 96)), ExampleCommand, i)
		go print(r.Read(), "\n")
	}
	return
}