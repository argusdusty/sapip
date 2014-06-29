package sapip

import (
	"log"
	"sync"
)

type SAIQueue struct {
	Lock              *sync.Mutex // Global lock
	ExecLock          *sync.Mutex // Lock for manipulating ExecElements
	EmptyCond         *sync.Cond  // Wait for queue to be non-empty
	LimitCond         *sync.Cond  // Wait for open slot in ExecElements
	Elements          IndexedElements
	ExecElements      []*Element
	SimultaneousLimit int
	Function          func(name string, data []string) string
	Stopped           bool
}

func (Q *SAIQueue) Init(SimultaneousLimit int, Function func(name string, data []string) string) {
	Q.Lock = new(sync.Mutex)
	Q.ExecLock = new(sync.Mutex)
	Q.EmptyCond = sync.NewCond(new(sync.Mutex))
	Q.LimitCond = sync.NewCond(new(sync.Mutex))
	Q.Elements = IndexedElements{make(map[string]*Element, 0), nil, nil}
	Q.ExecElements = make([]*Element, 0)
	Q.SimultaneousLimit = SimultaneousLimit
	Q.Function = Function
	Q.Stopped = false
}

func (Q *SAIQueue) AddElement(Name, Data string) SafeReturn {
	Q.Lock.Lock()
	defer Q.Lock.Unlock()
	SR := Q.Elements.AddElement(Name, Data)
	Q.EmptyCond.L.Lock()
	defer Q.EmptyCond.L.Unlock()
	Q.EmptyCond.Broadcast()
	return SR
}

func (Q *SAIQueue) Exec(e *Element) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Error in queue on element:", e.Name, "-", r)
		}
		Q.ExecLock.Lock()
		defer Q.ExecLock.Unlock()
		for i, elem := range Q.ExecElements {
			if elem == e {
				Q.ExecElements = append(Q.ExecElements[:i], Q.ExecElements[i+1:]...)
				break
			}
		}
		Q.LimitCond.L.Lock()
		defer Q.LimitCond.L.Unlock()
		Q.LimitCond.Broadcast()
	}()
	r := ""
	defer func() { e.OutChannel.Return(r) }()
	r = Q.Function(e.Name, e.Data)
}

func (Q *SAIQueue) Stop() {
	Q.Lock.Lock()
	defer Q.Lock.Unlock()
	Q.Stopped = true
}

func (Q *SAIQueue) Run() {
	for {
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
