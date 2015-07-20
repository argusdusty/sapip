// Copyright (C) 2015  Mark Canning
// Author: Argusdusty (Mark Canning)
// Email: argusdusty@gmail.com

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"github.com/argusdusty/sapip"
	"strings"
	"time"
)

type SElement struct {
	Name     string
	Data     string
	Priority int
}

const ExampleDelay = 100 * time.Millisecond
const ExampleSimultaneousLimit = 10

var ExampleData = []SElement{
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
	SElement{"8", "a", 1},
	SElement{"8", "b", 0},
	SElement{"8", "c", 2},
}

var FullDelay = ExampleDelay * time.Duration(len(ExampleData))

func ExampleCommand(name string, data []string) string {
	fmt.Println(time.Now(), name, strings.Join(data, " "), "Executing!")
	return name + " " + strings.Join(data, " ") + " Finished!"
}

var ExampleSAPIPQueue = sapip.NewSAPIPQueue(ExampleCommand, ExampleSimultaneousLimit)
var ExampleSAIPQueue = sapip.NewSAIPQueue(ExampleCommand, ExampleSimultaneousLimit)
var ExampleSAPIQueue = sapip.NewSAPIQueue(ExampleCommand, ExampleSimultaneousLimit)
var ExampleSAIQueue = sapip.NewSAIQueue(ExampleCommand, ExampleSimultaneousLimit)

func main() {
	go ExampleSAPIPQueue.Run(ExampleDelay)
	go ExampleSAIPQueue.Run()
	go ExampleSAPIQueue.Run(ExampleDelay)
	go ExampleSAIQueue.Run()
	fmt.Println(time.Now(), "Testing SAPIP queue")
	for _, e := range ExampleData {
		sr := ExampleSAPIPQueue.AddElement(e.Name, e.Data, e.Priority)
		fmt.Println(time.Now(), "Insert: Name:", e.Name, "Data:", e.Data, "Priority:", e.Priority)
		go func() { fmt.Println(time.Now(), "SAPIP:", sr.Read()) }()
	}
	time.Sleep(FullDelay)
	fmt.Println(time.Now(), "Testing SAIP queue")
	for _, e := range ExampleData {
		sr := ExampleSAIPQueue.AddElement(e.Name, e.Data, e.Priority)
		fmt.Println(time.Now(), "Insert: Name:", e.Name, "Data:", e.Data, "Priority:", e.Priority)
		go func() { fmt.Println(time.Now(), "SAIP:", sr.Read()) }()
	}
	time.Sleep(FullDelay)
	fmt.Println(time.Now(), "Testing SAPI queue")
	for _, e := range ExampleData {
		sr := ExampleSAPIQueue.AddElement(e.Name, e.Data)
		fmt.Println(time.Now(), "Insert: Name:", e.Name, "Data:", e.Data)
		go func() { fmt.Println(time.Now(), "SAPI:", sr.Read()) }()
	}
	time.Sleep(FullDelay)
	fmt.Println(time.Now(), "Testing SAI queue")
	for _, e := range ExampleData {
		sr := ExampleSAIQueue.AddElement(e.Name, e.Data)
		fmt.Println(time.Now(), "Insert: Name:", e.Name, "Data:", e.Data)
		go func() { fmt.Println(time.Now(), "SAI:", sr.Read()) }()
	}
	time.Sleep(FullDelay)
	return
}
