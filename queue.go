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
}

type DoubleSortedElements struct {
	NameIndex   map[string]int
	PriorityMap map[int][]Element
	Priorities  []int // In order
}

type Queue struct {
	Lock              *sync.Mutex
	ExecLock          *sync.Mutex
	Elements          DoubleSortedElements
	ExecElements      []Element
	SimultaneousLimit int
	Function          func(name string, data []string) string
	Stopped           bool
}

func (Q *Queue) Init(SimultaneousLimit int, Function func(name string, data []string) string) {
	Q.Lock = new(sync.Mutex)
	Q.ExecLock = new(sync.Mutex)
	Q.Elements = DoubleSortedElements{make(map[string]int, 0), make(map[int][]Element, 0), make([]int, 0)}
	Q.ExecElements = make([]Element, 0)
	Q.SimultaneousLimit = SimultaneousLimit
	Q.Function = Function
	Q.Stopped = false
}

func (D *DoubleSortedElements) AddPriority(Priority int) {
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
}

func (D *DoubleSortedElements) AddElement(Name, Data string, Priority int) SafeReturn {
	if p, ok := D.NameIndex[Name]; ok {
		elements := D.PriorityMap[p]
		j := 0
		for elements[j].Name != Name {
			j++
		}
		found := false
		for _, d := range elements[j].Data {
			if d == Data {
				found = true
				break
			}
		}
		if p > Priority {
			BaseData := elements[j].Data
			if !found {
				BaseData = append(BaseData, Data)
			}
			D.NameIndex[Name] = Priority
			OutChannel := elements[j].OutChannel
			elements = append(elements[:j], elements[j+1:]...)
			if a, ok := D.PriorityMap[Priority]; ok {
				D.PriorityMap[Priority] = append(a, Element{Name, BaseData, Priority, OutChannel})
			} else {
				D.AddPriority(Priority)
				D.PriorityMap[Priority] = []Element{Element{Name, BaseData, Priority, OutChannel}}
			}
			if len(elements) == 0 {
				i := 0
				j := len(D.Priorities)
				for i < j {
					h := (i + j) >> 1
					if D.Priorities[h] < p {
						i = h + 1
					} else {
						j = h
					}
				}
				D.Priorities = append(D.Priorities[:i], D.Priorities[i+1:]...)
				delete(D.PriorityMap, p)
			} else {
				D.PriorityMap[p] = elements
			}
			return OutChannel
		}
		if !found {
			elements[j].Data = append(elements[j].Data, Data)
		}
		return elements[j].OutChannel
	}
	e := Element{Name, []string{Data}, Priority, make(SafeReturn, 1)}
	D.NameIndex[Name] = Priority
	if a, ok := D.PriorityMap[Priority]; ok {
		D.PriorityMap[Priority] = append(a, e)
	} else {
		D.AddPriority(Priority)
		D.PriorityMap[Priority] = []Element{e}
	}
	return e.OutChannel
}

func (D *DoubleSortedElements) Pop() Element {
	elements := D.PriorityMap[D.Priorities[0]]
	e := elements[0]
	if len(elements) == 1 {
		delete(D.PriorityMap, D.Priorities[0])
		D.Priorities = D.Priorities[1:]
	} else {
		D.PriorityMap[D.Priorities[0]] = elements[1:]
	}
	delete(D.NameIndex, e.Name)
	return e
}

func (Q *Queue) AddElement(Name, Data string, Priority int) SafeReturn {
	Q.Lock.Lock()
	defer Q.Lock.Unlock()
	Q.ExecLock.Lock()
	defer Q.ExecLock.Unlock()
	for _, e := range Q.ExecElements {
		if Name == e.Name {
			for _, d := range e.Data {
				if d == Data {
					return e.OutChannel
				}
			}
		}
	}
	return Q.Elements.AddElement(Name, Data, Priority)
}

func (Q *Queue) Exec(e Element) {
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
	defer func() {
		if r := recover(); r != nil {
			log.Println("Queue runtime error:", r)
		}
	}()
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
