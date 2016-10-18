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

package sapip

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type SElement struct {
	Name     string
	Data     string
	Priority int
}

const (
	ExampleDelay             = 1000 * time.Millisecond
	ExampleSimultaneousLimit = 1
)

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

func ExampleCommand(name string, data []string) string {
	return name + " " + strings.Join(data, " ") + " Finished!"
}

func TestSapip(t *testing.T) {
	fmt.Println("Testing SAPIP queue")
	ExampleSAPIPQueue := NewSAPIPQueue(ExampleCommand, ExampleSimultaneousLimit)
	go ExampleSAPIPQueue.Run(ExampleDelay)
	wg := new(sync.WaitGroup)
	for _, e := range ExampleData {
		sr := ExampleSAPIPQueue.AddElement(e.Name, e.Data, e.Priority)
		fmt.Println("Insert: Name:", e.Name, "Data:", e.Data, "Priority:", e.Priority)
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("SAPIP:", sr.Read())
		}()
	}
	wg.Wait()
}

func TestSaip(t *testing.T) {
	fmt.Println("Testing SAIP queue")
	ExampleSAIPQueue := NewSAIPQueue(ExampleCommand, ExampleSimultaneousLimit)
	go ExampleSAIPQueue.Run()
	wg := new(sync.WaitGroup)
	for _, e := range ExampleData {
		sr := ExampleSAIPQueue.AddElement(e.Name, e.Data, e.Priority)
		fmt.Println("Insert: Name:", e.Name, "Data:", e.Data, "Priority:", e.Priority)
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("SAIP:", sr.Read())
		}()
	}
	wg.Wait()
}

func TestSapi(t *testing.T) {
	fmt.Println("Testing SAPI queue")
	ExampleSAPIQueue := NewSAPIQueue(ExampleCommand, ExampleSimultaneousLimit)
	go ExampleSAPIQueue.Run(ExampleDelay)
	wg := new(sync.WaitGroup)
	for _, e := range ExampleData {
		sr := ExampleSAPIQueue.AddElement(e.Name, e.Data)
		fmt.Println("Insert: Name:", e.Name, "Data:", e.Data)
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("SAPI:", sr.Read())
		}()
	}
	wg.Wait()
}

func TestSai(t *testing.T) {
	fmt.Println("Testing SAI queue")
	ExampleSAIQueue := NewSAIQueue(ExampleCommand, ExampleSimultaneousLimit)
	go ExampleSAIQueue.Run()
	wg := new(sync.WaitGroup)
	for _, e := range ExampleData {
		sr := ExampleSAIQueue.AddElement(e.Name, e.Data)
		fmt.Println("Insert: Name:", e.Name, "Data:", e.Data)
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("SAI:", sr.Read())
		}()
	}
	wg.Wait()
}

func BenchmarkSaipAddElements(b *testing.B) {
	wg := new(sync.WaitGroup)
	DrainCommand := func(name string, data []string) string {
		wg.Done()
		return ""
	}
	BenchSAIPQueue := NewSAIPQueue(DrainCommand, 1)
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		BenchSAIPQueue.AddElement(strconv.Itoa(i), "", 0)
	}
	go BenchSAIPQueue.Run()
	wg.Wait()
}

func BenchmarkSaipUpdatePriority(b *testing.B) {
	wg := new(sync.WaitGroup)
	DrainCommand := func(name string, data []string) string {
		wg.Done()
		return ""
	}
	BenchSAIPQueue := NewSAIPQueue(DrainCommand, 1)
	wg.Add(1)
	for i := 0; i < b.N; i++ {
		BenchSAIPQueue.AddElement("", "", i)
	}
	go BenchSAIPQueue.Run()
	wg.Wait()
}

func BenchmarkSaiAddElements(b *testing.B) {
	wg := new(sync.WaitGroup)
	DrainCommand := func(name string, data []string) string {
		wg.Done()
		return ""
	}
	BenchSAIQueue := NewSAIQueue(DrainCommand, 1)
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		BenchSAIQueue.AddElement(strconv.Itoa(i), "")
	}
	go BenchSAIQueue.Run()
	wg.Wait()
}
