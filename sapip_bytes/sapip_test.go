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

// sapip_bytes contains experimental changes to use byte arrays instead of strings
// Might be faster if you manage your memory well and are currently converting
// byte arrays to strings for sapip keys
package sapip_bytes

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
	"time"
)

type SElement struct {
	Name     []byte
	Data     []byte
	Priority int
}

const (
	ExampleDelay             = 1000 * time.Millisecond
	ExampleSimultaneousLimit = 1
)

var ExampleData = []SElement{
	SElement{[]byte("1"), []byte("a"), 2},
	SElement{[]byte("1"), []byte("b"), 3},
	SElement{[]byte("2"), []byte("a"), 2},
	SElement{[]byte("2"), []byte("a"), 1},
	SElement{[]byte("3"), []byte("a"), 2},
	SElement{[]byte("3"), []byte("a"), 3},
	SElement{[]byte("4"), []byte("a"), 1},
	SElement{[]byte("4"), []byte("a"), 1},
	SElement{[]byte("5"), []byte("a"), 2},
	SElement{[]byte("5"), []byte("b"), 2},
	SElement{[]byte("6"), []byte("b"), 1},
	SElement{[]byte("6"), []byte("a"), 1},
	SElement{[]byte("7"), []byte("b"), 1},
	SElement{[]byte("7"), []byte("a"), 0},
	SElement{[]byte("8"), []byte("a"), 1},
	SElement{[]byte("8"), []byte("b"), 0},
	SElement{[]byte("8"), []byte("c"), 2},
}

func ExampleCommand(name []byte, data [][]byte) []byte {
	return bytes.Join(append(append([][]byte{name}, data...), []byte("Finished!")), []byte{' '})
}

func TestSapip(t *testing.T) {
	fmt.Println("Testing SAPIP queue")
	ExampleSAPIPQueue := NewSAPIPQueue(ExampleCommand, ExampleSimultaneousLimit)
	go ExampleSAPIPQueue.Run(ExampleDelay)
	wg := new(sync.WaitGroup)
	for _, e := range ExampleData {
		sr := ExampleSAPIPQueue.AddElement(e.Name, e.Data, e.Priority)
		fmt.Println("Insert: Name:", string(e.Name), "Data:", string(e.Data), "Priority:", e.Priority)
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("SAPIP:", string(sr.Read()))
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
		fmt.Println("Insert: Name:", string(e.Name), "Data:", string(e.Data), "Priority:", e.Priority)
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("SAIP:", string(sr.Read()))
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
		fmt.Println("Insert: Name:", string(e.Name), "Data:", string(e.Data))
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("SAPI:", string(sr.Read()))
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
		fmt.Println("Insert: Name:", string(e.Name), "Data:", string(e.Data))
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("SAI:", string(sr.Read()))
		}()
	}
	wg.Wait()
}

func Uint32ToByteArray(i uint32) []byte {
	counter := make([]byte, 4)
	counter[0] = byte(i >> 24)
	counter[1] = byte(i >> 16)
	counter[2] = byte(i >> 8)
	counter[3] = byte(i)
	return counter
}

func BenchmarkSaipAddElements(b *testing.B) {
	wg := new(sync.WaitGroup)
	DrainCommand := func(name []byte, data [][]byte) []byte {
		wg.Done()
		return nil
	}
	BenchSAIPQueue := NewSAIPQueue(DrainCommand, 1)
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		BenchSAIPQueue.AddElement(Uint32ToByteArray(uint32(i)), nil, 0)
	}
	go BenchSAIPQueue.Run()
	wg.Wait()
}

func BenchmarkSaipUpdatePriority(b *testing.B) {
	wg := new(sync.WaitGroup)
	DrainCommand := func(name []byte, data [][]byte) []byte {
		wg.Done()
		return nil
	}
	BenchSAIPQueue := NewSAIPQueue(DrainCommand, 1)
	wg.Add(1)
	for i := 0; i < b.N; i++ {
		BenchSAIPQueue.AddElement([]byte{}, nil, i)
	}
	go BenchSAIPQueue.Run()
	wg.Wait()
}

func BenchmarkSaiAddElements(b *testing.B) {
	wg := new(sync.WaitGroup)
	DrainCommand := func(name []byte, data [][]byte) []byte {
		wg.Done()
		return nil
	}
	BenchSAIQueue := NewSAIQueue(DrainCommand, 1)
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		BenchSAIQueue.AddElement(Uint32ToByteArray(uint32(i)))
	}
	go BenchSAIQueue.Run()
	wg.Wait()
}
