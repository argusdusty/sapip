package main

import (
	"github.com/argusdusty/sapip"
	"time"
)

ExampleQueue = new(sapip.Queue)
ExampleDelay = 100*time.Millisecond
ExampleSimultaneousLimit = 100

func init() {
	ExampleQueue.Run(ExampleDelay, ExampleSimultaneousLimit)
}

func main() {
	ExampleCommand := func(input string) { return input + " Done!" }
	for i := 10; i > 0; i-- {
		r := ExampleQueue.AddElement("Testing: " + string(byte(i + 96)))
		go print(r.Read(), "\n")
	}
	return
}