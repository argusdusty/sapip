package sapip

import (
	"sync"
	"time"
)

type Queue struct {
	Lock              *sync.Mutex // Global lock
	ExecLock          *sync.Mutex // Lock for manipulating ExecElements
	EmptyCond         *sync.Cond  // Wait for queue to be non-empty
	LimitCond         *sync.Cond  // Wait for open slot in ExecElements
	Elements          IndexedPriorityElements
	ExecElements      []*PriorityElement
	SimultaneousLimit int
	Function          func(name string, data []string) string
	Stopped           bool
}

type SAPIPQueue Queue

func (Q *Queue) Init(SimultaneousLimit int, Function func(name string, data []string) string) {
	Q.Lock = new(sync.Mutex)
	Q.ExecLock = new(sync.Mutex)
	Q.EmptyCond = sync.NewCond(new(sync.Mutex))
	Q.LimitCond = sync.NewCond(new(sync.Mutex))
	Q.Elements = IndexedPriorityElements{make(map[string]*PriorityElement, 0), make(map[int]*PriorityElement, 0), make([]int, 0), make(map[int]int, 0), nil}
	Q.ExecElements = make([]*PriorityElement, 0)
	Q.SimultaneousLimit = SimultaneousLimit
	Q.Function = Function
	Q.Stopped = false
}

func (Q *Queue) AddElement(Name, Data string, Priority int) SafeReturn {
	Q.Lock.Lock()
	defer Q.Lock.Unlock()
	// Add the element
	SR := Q.Elements.AddElement(Name, Data, Priority)
	// Broadcast that the queue might be non-empty
	Q.EmptyCond.L.Lock()
	defer Q.EmptyCond.L.Unlock()
	Q.EmptyCond.Broadcast()
	return SR
}

func (Q *Queue) Exec(e *PriorityElement) {
	defer func() {
		if r := recover(); r != nil {
			logFunc("Error in queue on element:", e.Name, "-", r)
		}
		// Remove the element
		Q.ExecLock.Lock()
		defer Q.ExecLock.Unlock()
		for i, elem := range Q.ExecElements {
			if elem == e {
				Q.ExecElements = append(Q.ExecElements[:i], Q.ExecElements[i+1:]...)
				break
			}
		}
		// Broadcast the now empty slot in ExecElements
		Q.LimitCond.L.Lock()
		defer Q.LimitCond.L.Unlock()
		Q.LimitCond.Broadcast()
	}()
	// Execute the function and return it in a defer (in case it panics)
	r := ""
	defer func() { e.OutChannel.Return(r) }()
	r = Q.Function(e.Name, e.Data)
}

func (Q *Queue) Stop() {
	Q.Lock.Lock()
	defer Q.Lock.Unlock()
	Q.Stopped = true
}

func (Q *Queue) Run(Wait time.Duration) {
	a := time.Tick(Wait)
	for _ = range a {
		if Q.Stopped {
			break
		}
		Q.EmptyCond.L.Lock()
		for Q.Elements.Front == nil {
			Q.EmptyCond.Wait()
		}
		Q.EmptyCond.L.Unlock()
		Q.LimitCond.L.Lock()
		for len(Q.ExecElements) >= Q.SimultaneousLimit {
			Q.LimitCond.Wait()
		}
		Q.LimitCond.L.Unlock()
		Q.Lock.Lock()
		e := Q.Elements.Pop()
		Q.Lock.Unlock()
		go func() {
			var sr SafeReturn
			for {
				found := false
				Q.ExecLock.Lock()
				for _, elem := range Q.ExecElements {
					if elem.Name == e.Name && elem != e {
						found = true
						sr = elem.OutChannel
					}
				}
				Q.ExecLock.Unlock()
				if !found {
					break
				}
				sr.Read()
			}
			Q.ExecLock.Lock()
			Q.ExecElements = append(Q.ExecElements, e)
			go Q.Exec(e)
			Q.ExecLock.Unlock()
		}()
	}
}
