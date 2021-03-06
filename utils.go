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
	"log"
)

type SafeReturn chan string

func (SR SafeReturn) Return(value string) { SR <- value }
func (SR SafeReturn) Read() string        { value := <-SR; SR <- value; return value }

type Element struct {
	Name       string
	Data       []string
	OutChannel SafeReturn
	Next       *Element
	Prev       *Element
}

type PriorityElement struct {
	Name       string
	Data       []string
	Priority   int
	OutChannel SafeReturn
	Next       *PriorityElement
	Prev       *PriorityElement
}

type QueueFunction func(name string, data []string) string
type QueueErrFunction func(name string, err interface{})

func defaultErrFunc(name string, err interface{}) {
	log.Println("Error in queue on element:", name, "-", err)
}

// Map Queue to SAPIPQueue
type Queue SAPIPQueue

var NewQueue = NewSAPIPQueue

type IndexedElements struct {
	NameIndex map[string]*Element // Map from each name to pointer to corresponding element
	Front     *Element            // Front element
	End       *Element            // Last element
}

func MakeIndexedElements() IndexedElements {
	return IndexedElements{make(map[string]*Element), nil, nil}
}

// Insert an element
func (D *IndexedElements) AddElement(Name string, Data ...string) SafeReturn {
	if p, ok := D.NameIndex[Name]; ok {
		p.Data = append(p.Data, Data...)
		return p.OutChannel
	}
	e := &Element{Name, Data, make(SafeReturn, 1), nil, nil}
	if D.End != nil {
		D.End.Next = e
		e.Prev = D.End
	}
	if D.Front == nil {
		D.Front = e
	}
	D.End = e
	D.NameIndex[e.Name] = e
	return e.OutChannel
}

// Remove an element
func (D *IndexedElements) RemoveElement(e *Element) {
	if e.Prev != nil {
		e.Prev.Next = e.Next
	}
	if e.Next != nil {
		e.Next.Prev = e.Prev
	}
	if D.Front == e {
		D.Front = e.Next
	}
	if D.End == e {
		D.End = e.Prev
	}
	e.Next = nil
	e.Prev = nil
	delete(D.NameIndex, e.Name)
}

// Remove the front element
func (D *IndexedElements) Pop() *Element {
	e := D.Front
	if e.Next != nil {
		e.Next.Prev = nil
	}
	D.Front = e.Next
	if D.End == e {
		D.End = nil
	}
	e.Next = nil
	e.Prev = nil
	delete(D.NameIndex, e.Name)
	return e
}

// Removes all elements into a slice
// Equivalent to appending all D.Pop() values into an array
func (D *IndexedElements) DumpElements() []*Element {
	r := make([]*Element, 0, len(D.NameIndex))
	for _, v := range D.NameIndex {
		r = append(r, v)
	}
	D.NameIndex = make(map[string]*Element)
	D.Front = nil
	D.End = nil
	return r
}

type IndexedPriorityElements struct {
	NameIndex      map[string]*PriorityElement // Map from each name to pointer to corresponding element
	PriorityMap    map[int]*PriorityElement    // Map from each priority to the element which is at the end of that priority
	Priorities     []int                       // List of priorities in sorted order
	PriorityLength map[int]int                 // Map from each priority to the number of elements which have the priority
	Front          *PriorityElement            // Front element
}

func MakeIndexedPriorityElements() IndexedPriorityElements {
	return IndexedPriorityElements{make(map[string]*PriorityElement), make(map[int]*PriorityElement), make([]int, 0), make(map[int]int), nil}
}

func (D *IndexedPriorityElements) addPriority(Priority int) int {
	// Binary search to determine the index to insert Priority
	i := 0
	j := len(D.Priorities)
	for i < j {
		h := (i + j) >> 1
		if D.Priorities[h] < Priority {
			i = h + 1
		} else {
			j = h
		}
	}
	// Insert it
	D.Priorities = append(D.Priorities[:i], append([]int{Priority}, D.Priorities[i:]...)...)
	// Return the index
	return i
}

func (D *IndexedPriorityElements) add(e *PriorityElement) {
	if a, ok := D.PriorityMap[e.Priority]; ok {
		// If we already have the priority, all we need to do is put the new element at the end
		if a.Next != nil {
			e.Next = a.Next
		}
		a.Next = e
		e.Prev = a
		D.PriorityLength[e.Priority] += 1
	} else {
		// Otherwise we need to create a new priority
		i := D.addPriority(e.Priority)
		// i is the number of priorities > e.Priority
		if i == 0 {
			// e has the smallest priority and goes to the front
			e.Next = D.Front
			D.Front = e
		} else {
			// e Needs to be placed between two priorities
			x := D.PriorityMap[D.Priorities[i-1]]
			e.Next = x.Next
			x.Next = e
			e.Prev = x
		}
		D.PriorityLength[e.Priority] = 1
	}
	// Add e to the indexes
	D.PriorityMap[e.Priority] = e
	D.NameIndex[e.Name] = e
}

// Insert an element
func (D *IndexedPriorityElements) AddElement(Name, Data string, Priority int) SafeReturn {
	// If the element name is already in the queue we need to do special stuff
	if p, ok := D.NameIndex[Name]; ok {
		// Append the data
		p.Data = append(p.Data, Data)
		// If the new priority is smaller, we need to remove the old element and insert it with the new priority
		if p.Priority > Priority {
			// Note to self: Possibility of memory leak in re-using p.Data/p.OutChannel?
			e := &PriorityElement{Name, p.Data, Priority, p.OutChannel, nil, nil}
			D.RemoveElement(p)
			D.add(e)
		}
		return p.OutChannel
	}
	// Go ahead and insert the element
	e := &PriorityElement{Name, []string{Data}, Priority, make(SafeReturn, 1), nil, nil}
	D.add(e)
	return e.OutChannel
}

// Remove an element
func (D *IndexedPriorityElements) RemoveElement(e *PriorityElement) {
	// First, reorder the pointers
	if e.Prev != nil {
		e.Prev.Next = e.Next
	}
	if e.Next != nil {
		e.Next.Prev = e.Prev
	}
	if D.Front == e {
		D.Front = e.Next
	}
	if D.PriorityLength[e.Priority] == 1 {
		// e is the only element of that priority, so we need to delete the priority
		// Start with a binary search to determine the index of the priority
		// Note to self: This might just be faster as a linear search. How many distinct priorities will there be?
		i := 0
		j := len(D.Priorities)
		for i < j {
			h := (i + j) >> 1
			if D.Priorities[h] < e.Priority {
				i = h + 1
			} else {
				j = h
			}
		}
		// And delete the priority
		D.Priorities = append(D.Priorities[:i], D.Priorities[i+1:]...)
		delete(D.PriorityMap, e.Priority)
		delete(D.PriorityLength, e.Priority)
	} else {
		// Just remove e from the priority
		D.PriorityLength[e.Priority] -= 1
		if D.PriorityMap[e.Priority] == e {
			D.PriorityMap[e.Priority] = e.Prev
		}
	}
	// Clear the pointers of e
	e.Next = nil
	e.Prev = nil
	delete(D.NameIndex, e.Name)
}

// Remove the front element
func (D *IndexedPriorityElements) Pop() *PriorityElement {
	e := D.Front
	// Set the front to the next element and clear the next element's pointer to e
	if e.Next != nil {
		e.Next.Prev = nil
	}
	D.Front = e.Next
	// Decrement the priority/priority length maps
	if D.PriorityLength[e.Priority] == 1 {
		D.Priorities = D.Priorities[1:]
		delete(D.PriorityLength, e.Priority)
		delete(D.PriorityMap, e.Priority)
	} else {
		D.PriorityLength[e.Priority] -= 1
	}
	// Clear the pointers of e
	e.Next = nil
	e.Prev = nil
	// Remove e
	delete(D.NameIndex, e.Name)
	return e
}

// Removes all elements into a slice
// Equivalent to appending all D.Pop() values into an array
func (D *IndexedPriorityElements) DumpElements() []*PriorityElement {
	r := make([]*PriorityElement, 0, len(D.NameIndex))
	for _, v := range D.NameIndex {
		r = append(r, v)
	}
	D.NameIndex = make(map[string]*PriorityElement)
	D.PriorityMap = make(map[int]*PriorityElement)
	D.Priorities = make([]int, 0)
	D.Front = nil
	return r
}
