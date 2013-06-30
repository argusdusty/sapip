package sapip

import (
	"log"
	"sync"
	"time"
)

type SafeReturn chan string

func (SR SafeReturn) Return(value string) { SR <- value }
func (SR SafeReturn) Read() string        { value := <-SR; SR <- value; return value }

type Element struct {
	Name       string
	Data       []string
	Priority   int
	OutChannel SafeReturn
	Prev       *Element
	Next       *Element
}

type DoubleSortedElements struct {
	NameIndex      map[string]*Element // Map from each name to pointer to corresponding element
	PriorityMap    map[int]*Element    // Map from each priority to the element which is at the end of that priority
	Priorities     []int               // List of priorities in sorted order
	PriorityLength map[int]int         // Map from eac priority to the number of elements which have the priority
	Front          *Element            // Front element
}

type Queue struct {
	Lock              *sync.Mutex
	ExecLock          *sync.Mutex
	Elements          DoubleSortedElements
	ExecElements      []*Element
	SimultaneousLimit int
	Function          func(name string, data []string) string
	Stopped           bool
}

func (Q *Queue) Init(SimultaneousLimit int, Function func(name string, data []string) string) {
	Q.Lock = new(sync.Mutex)
	Q.ExecLock = new(sync.Mutex)
	Q.Elements = DoubleSortedElements{make(map[string]*Element, 0), make(map[int]*Element, 0), make([]int, 0), make(map[int]int, 0), nil}
	Q.ExecElements = make([]*Element, 0)
	Q.SimultaneousLimit = SimultaneousLimit
	Q.Function = Function
	Q.Stopped = false
}

func (D *DoubleSortedElements) AddPriority(Priority int) int {
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
	D.Priorities = append(D.Priorities[:i], append([]int{Priority}, D.Priorities[i:]...)...)
	return i
}

func (D *DoubleSortedElements) Remove(e *Element) {
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
		D.Priorities = append(D.Priorities[:i], D.Priorities[i+1:]...)
		delete(D.PriorityMap, e.Priority)
		delete(D.PriorityLength, e.Priority)
	} else {
		D.PriorityLength[e.Priority] -= 1
		if D.PriorityMap[e.Priority] == e {
			D.PriorityMap[e.Priority] = e.Prev
		}
	}
	e.Next = nil
	e.Prev = nil
	delete(D.NameIndex, e.Name)
}

func (D *DoubleSortedElements) Add(e *Element) {
	if a, ok := D.PriorityMap[e.Priority]; ok {
		if a.Next != nil {
			e.Next = a.Next
		}
		a.Next = e
		e.Prev = a
		D.PriorityLength[e.Priority] += 1
	} else {
		i := D.AddPriority(e.Priority)
		if i == 0 {
			e.Next = D.Front
			D.Front = e
		} else {
			x := D.PriorityMap[D.Priorities[i-1]]
			e.Next = x.Next
			x.Next = e
			e.Prev = x
		}
		D.PriorityLength[e.Priority] = 1
	}
	D.PriorityMap[e.Priority] = e
	D.NameIndex[e.Name] = e
}

func (D *DoubleSortedElements) AddElement(Name, Data string, Priority int) SafeReturn {
	if p, ok := D.NameIndex[Name]; ok {
		found := false
		for _, d := range p.Data {
			if d == Data {
				found = true
				break
			}
		}
		if p.Priority > Priority {
			BaseData := p.Data
			if !found {
				BaseData = append(BaseData, Data)
			}
			OutChannel := p.OutChannel
			e := &Element{Name, BaseData, Priority, OutChannel, nil, nil}
			D.Remove(p)
			D.Add(e)
		}
		if !found {
			p.Data = append(p.Data, Data)
		}
		return p.OutChannel
	}
	e := &Element{Name, []string{Data}, Priority, make(SafeReturn, 1), nil, nil}
	D.Add(e)
	return e.OutChannel
}

func (D *DoubleSortedElements) Pop() *Element {
	e := D.Front
	if e.Next != nil {
		e.Next.Prev = e.Prev
	}
	D.Front = e.Next
	if D.PriorityLength[e.Priority] == 1 {
		D.Priorities = D.Priorities[1:]
		delete(D.PriorityLength, e.Priority)
		delete(D.PriorityMap, e.Priority)
	} else {
		D.PriorityLength[e.Priority] -= 1
	}
	e.Next = nil
	e.Prev = nil
	delete(D.NameIndex, e.Name)
	return e
}

func (Q *Queue) AddElement(Name, Data string, Priority int) SafeReturn {
	Q.Lock.Lock()
	defer Q.Lock.Unlock()
	return Q.Elements.AddElement(Name, Data, Priority)
}

func (Q *Queue) Exec(e *Element) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Error in queue on element:", e.Name, "-", r)
		}
		Q.ExecLock.Lock()
		defer Q.ExecLock.Unlock()
		for i := len(Q.ExecElements) - 1; i >= 0; i-- {
			elem := Q.ExecElements[i]
			if elem.Name == e.Name {
				Q.ExecElements = append(Q.ExecElements[:i], Q.ExecElements[i+1:]...)
				break
			}
		}
	}()
	r := ""
	defer func() { e.OutChannel.Return(r) }()
	r = Q.Function(e.Name, e.Data)
}

func (Q *Queue) Stop() {
	Q.Lock.Lock()
	defer Q.Lock.Unlock()
	Q.ExecLock.Lock()
	defer Q.ExecLock.Unlock()
	Q.Stopped = true
}

func (Q *Queue) Run(Wait time.Duration) {
	a := time.Tick(Wait)
	for _ = range a {
		if Q.Stopped {
			break
		}
		Q.Lock.Lock()
		if len(Q.Elements.Priorities) > 0 && len(Q.ExecElements) < Q.SimultaneousLimit {
			e := Q.Elements.Pop()
			Q.ExecLock.Lock()
			Q.ExecElements = append(Q.ExecElements, e)
			Q.ExecLock.Unlock()
			go Q.Exec(e)
		}
		Q.Lock.Unlock()
	}
}
