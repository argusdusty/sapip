package main

import (
	"github.com/argusdusty/sapip"
	"time"
	"strconv"
)

var ExampleQueue = new(sapip.Queue)
const ExampleDelay = 1000*time.Millisecond
const ExampleSimultaneousLimit = 100

func init() {
	ExampleQueue.Init(ExampleDelay, ExampleSimultaneousLimit)
}

func main() {
	ExampleCommand := func(input string) string { return input + " Done!" }
	for i := 10; i > 0; i-- {
		r, index := ExampleQueue.AddElement("Testing: " + string(byte(i + 96)), ExampleCommand, i)
		print("Insert: " + string(byte(i + 96)), " at position: " + strconv.Itoa(index), " \n")
		go func() { print(r.Read(), "\n") }()
	}
	r, index := ExampleQueue.AddElement("Testing: Error", func(input string) { panic(input); ExampleCommand(input) }, 5)
	print("Insert: Error", " at position: " + strconv.Itoa(index), " \n")
	go func() { print(r.Read(), "\n") }()
	time.Sleep(10*time.Second)
	return
}