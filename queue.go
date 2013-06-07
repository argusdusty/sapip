package sapip

import (
	"log"
	"sort"
	"sync"
	"time"
)

// Allows multiple threads to read a single value from this with .Read(), which blocks until a return value is sent
type SafeReturn chan string

func (SR SafeReturn) Return(value string) { SR <- value }
func (SR SafeReturn) Read() string        { value := <-SR; SR <- value; return value }

type ReducedElement struct {
	Priority   int
	OutChannel SafeReturn
}

type Element struct {
	Name       string
	Data       []string
	Priority   int
	OutChannel SafeReturn
}

type DoubleSortedElements struct {
	NameIndex      map[string]int
	PrioritySorted []Element
}

type Queue struct {
	Lock              *sync.Mutex
	ExecLock          *sync.Mutex
	Elements          DoubleSortedElements
	ExecElements      []Element
	SimultaneousLimit int
	Function          func(name string, data []string) string
	stop              bool
}

func (Q *Queue) InitAndRun(Time time.Duration, SimultaneousLimit int, Function func(name string, data []string) string) {
	Q.Init(SimultaneousLimit, Function)
	Q.Run(Time)
}

func (Q *Queue) Init(SimultaneousLimit int, Function func(name string, data []string) string) {
	Q.Lock = new(sync.Mutex)
	Q.ExecLock = new(sync.Mutex)
	Q.Elements = DoubleSortedElements{make(map[string]int, 0), make([]Element, 0)}
	Q.ExecElements = make([]Element, 0)
	Q.SimultaneousLimit = SimultaneousLimit
	Q.Function = Function
	Q.stop = false
}

func (D *DoubleSortedElements) AddElement(Name, Data string, Priority int) (SafeReturn, int) {
	p, ok := D.NameIndex[Name]
	if ok {
		j := sort.Search(len(D.PrioritySorted), func(j int) bool { return D.PrioritySorted[j].Priority >= p })
		for D.PrioritySorted[j].Name != Name {
			j++
		}
		found := false
		for _, d := range D.PrioritySorted[j].Data {
			if d == Data {
				found = true
				break
			}
		}
		if p > Priority {
			BaseData := D.PrioritySorted[j].Data
			if !found {
				BaseData = append(BaseData, Data)
			}
			D.NameIndex[Name] = Priority
			D.PrioritySorted = append(D.PrioritySorted[:j], D.PrioritySorted[j+1:]...)
			k := sort.Search(len(D.PrioritySorted), func(k int) bool { return D.PrioritySorted[k].Priority > Priority })
			e := Element{Name, BaseData, Priority, make(SafeReturn, 1)}
			D.PrioritySorted = append(D.PrioritySorted[:k], append([]Element{e}, D.PrioritySorted[k:]...)...)
			return e.OutChannel, k + 1
		}
		if !found {
			D.PrioritySorted[j].Data = append(D.PrioritySorted[j].Data, Data)
		}
		return D.PrioritySorted[j].OutChannel, j + 1
	}
	j := sort.Search(len(D.PrioritySorted), func(j int) bool { return D.PrioritySorted[j].Priority > Priority })
	e := Element{Name, []string{Data}, Priority, make(SafeReturn, 1)}
	D.PrioritySorted = append(D.PrioritySorted[:j], append([]Element{e}, D.PrioritySorted[j:]...)...)
	D.NameIndex[Name] = Priority
	return e.OutChannel, j + 1
}

func (D *DoubleSortedElements) Pop() Element {
	e := D.PrioritySorted[0]
	D.PrioritySorted = D.PrioritySorted[1:]
	delete(D.NameIndex, e.Name)
	return e
}

func (Q *Queue) AddElement(Name, Data string, Priority int) (SafeReturn, int) {
	Q.Lock.Lock()
	defer Q.Lock.Unlock()
	Q.ExecLock.Lock()
	defer Q.ExecLock.Unlock()
	for _, e := range Q.ExecElements {
		if Name == e.Name {
			for _, d := range e.Data {
				if d == Data {
					return e.OutChannel, 0
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

func (Q *Queue) Run(Wait time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Queue runtime error:", r)
		}
	}()
	a := time.Tick(Wait)
	for _ = range a {
		if Q.stop {
			break
		}
		Q.Lock.Lock()
		if len(Q.Elements.PrioritySorted) > 0 && len(Q.ExecElements) < Q.SimultaneousLimit {
			e := Q.Elements.Pop()
			Q.ExecLock.Lock()
			Q.ExecElements = append(Q.ExecElements, e)
			Q.ExecLock.Unlock()
			go Q.Exec(e)
		}
		Q.Lock.Unlock()
	}
}
