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
	"sync"
)

type SAIQueue struct {
	lock         *sync.Mutex // Global lock
	execLock     *sync.Mutex // Lock for manipulating execElements
	emptyCond    *sync.Cond  // Wait for queue to be non-empty
	limitCond    *sync.Cond  // Wait for open slot in execElements
	elements     IndexedElements
	execElements []*Element
	limit        int
	function     QueueFunction
	stopped      bool
	errFunc      QueueErrFunction
}

// Returns a new Safe Asynchronous Indexed Queue
// with f as the handler function for elements, and limit as the
// maximum number of simultaneously executing elements
func NewSAIQueue(f QueueFunction, limit int) *SAIQueue {
	var Q SAIQueue
	Q.lock = new(sync.Mutex)
	Q.execLock = new(sync.Mutex)
	Q.emptyCond = sync.NewCond(new(sync.Mutex))
	Q.limitCond = sync.NewCond(new(sync.Mutex))
	Q.elements = MakeIndexedElements()
	Q.execElements = make([]*Element, 0)
	Q.limit = limit
	Q.function = f
	Q.stopped = false
	Q.errFunc = defaultErrFunc
	return &Q
}

func (Q *SAIQueue) exec(e *Element) {
	defer func() {
		if r := recover(); r != nil {
			Q.errFunc(e.Name, r)
		}
		// Remove the element
		Q.execLock.Lock()
		defer Q.execLock.Unlock()
		for i, elem := range Q.execElements {
			if elem == e {
				Q.execElements = append(Q.execElements[:i], Q.execElements[i+1:]...)
				break
			}
		}
		// Broadcast the now empty slot in execElements
		Q.limitCond.L.Lock()
		defer Q.limitCond.L.Unlock()
		Q.limitCond.Broadcast()
	}()
	// Execute the function and return it in a defer (in case it panics)
	r := ""
	defer func() { e.OutChannel.Return(r) }()
	r = Q.function(e.Name, e.Data)
}

func (Q *SAIQueue) getTopElement() *Element {
	e := Q.elements.Front
	for e != nil {
		found := false
		Q.execLock.Lock()
		for _, elem := range Q.execElements {
			if elem.Name == e.Name && elem != e {
				found = true
			}
		}
		Q.execLock.Unlock()
		if !found {
			Q.lock.Lock()
			defer Q.lock.Unlock()
			if e == Q.elements.Front {
				Q.elements.Pop()
			} else {
				Q.elements.RemoveElement(e)
			}
			return e
		}
		e = e.Next
	}
	return nil
}

// Insert an element into the queue. If an element of that name already
// exists, the data will be appended into a list.
func (Q *SAIQueue) AddElement(Name, Data string) SafeReturn {
	Q.lock.Lock()
	defer Q.lock.Unlock()
	// Add the element
	sr := Q.elements.AddElement(Name, Data)
	// Broadcast that the queue might be non-empty
	Q.emptyCond.L.Lock()
	defer Q.emptyCond.L.Unlock()
	Q.emptyCond.Broadcast()
	return sr
}

// Update the limit on the number of simultaneously executing
// elements. If there are more than limit currently executing,
// the queue will wait until it is under the new limit.
func (Q *SAIQueue) SetLimit(limit int) {
	Q.lock.Lock()
	defer Q.lock.Unlock()
	Q.limitCond.L.Lock()
	defer Q.limitCond.L.Unlock()
	Q.limit = limit
	Q.limitCond.Broadcast()
}

// Set a new error handling function, which handles panics encountered
// When executing elements. By default this is a log.Println
func (Q *SAIQueue) SetErrorFunc(errFunc QueueErrFunction) {
	Q.lock.Lock()
	defer Q.lock.Unlock()
	Q.errFunc = errFunc
}

// Stops the execution of the queue. Currently executing elements
// will continue to run, but no new elements will start executing
func (Q *SAIQueue) Stop() {
	Q.lock.Lock()
	defer Q.lock.Unlock()
	Q.stopped = true
}

// Returns the number of elements waiting in the queue, and
// the number of currently executing elements
func (Q *SAIQueue) NumElements() (num int, execNum int) {
	Q.lock.Lock()
	Q.execLock.Lock()
	defer Q.execLock.Unlock()
	defer Q.lock.Unlock()
	num = len(Q.elements.NameIndex)
	execNum = len(Q.execElements)
	return
}

// Run the queue, executing elements repeatedly
// Will loop forever (until stopped), so spawn this in a new thread
func (Q *SAIQueue) Run() {
	for {
		if Q.stopped {
			break
		}
		// Wait for non-empty queue
		Q.emptyCond.L.Lock()
		for Q.elements.Front == nil {
			Q.emptyCond.Wait()
		}
		Q.emptyCond.L.Unlock()
		// Wait for an open space and wait for an element with a
		// name that doesn't match any currently executing elements
		Q.limitCond.L.Lock()
		var e *Element
		for {
			if len(Q.execElements) >= Q.limit {
				Q.limitCond.Wait()
			} else {
				e = Q.getTopElement()
				if e != nil {
					break
				}
				Q.limitCond.Wait()
			}
		}
		Q.limitCond.L.Unlock()
		Q.execLock.Lock()
		Q.execElements = append(Q.execElements, e)
		go Q.exec(e)
		Q.execLock.Unlock()
	}
}
