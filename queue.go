package sapip

import (
	"sync"
	"time"
	"sort"
)

type Command func(value string) string

type Element struct {
	Name string
	Function Command
	Priority int
	OutChannel SafeReturn
}

type DoubleSortedElements struct {
	NameIndex map[string]int
	PrioritySorted []Element
}

type Queue struct {
	Lock *sync.Mutex
	ExecLock *sync.Mutex
	Elements DoubleSortedElements
	ExecElements []Element
	Wait time.Duration
	SimultaneousLimit int
}

type SafeReturn chan string // Allows multiple threads to read from this with .Read(), which blocks until a return value is sent
func (SR SafeReturn) Return(value string) { SR <- value }
func (SR SafeReturn) Read() string { value := <- SR; SR <- value; return value }

func (D *DoubleSortedElements) AddElement(e Element) (SafeReturn, int) {
	p, ok := D.NameIndex[e.Name]
	if ok {
		j := sort.Search(len(D.PrioritySorted), func(j int) bool { return D.PrioritySorted[j].Priority >= p })
		for D.PrioritySorted[j].Name != e.Name { j++ }
		if p > e.Priority {
			D.NameIndex[e.Name] = e.Priority
			D.PrioritySorted = append(D.PrioritySorted[:j], D.PrioritySorted[j+1:]...)
			k := sort.Search(len(D.PrioritySorted), func(k int) bool { return D.PrioritySorted[k].Priority > e.Priority })
			D.PrioritySorted = append(D.PrioritySorted[:k], append([]Element{e}, D.PrioritySorted[k:]...)...)
			return e.OutChannel, k + 1
		}
		return D.PrioritySorted[j].OutChannel, j + 1
	}
	j := sort.Search(len(D.PrioritySorted), func(j int) bool { return D.PrioritySorted[j].Priority > e.Priority })
	D.PrioritySorted = append(D.PrioritySorted[:j], append([]Element{e}, D.PrioritySorted[j:]...)...)
	D.NameIndex[e.Name] = e.Priority
	return e.OutChannel, j + 1
}

func (D *DoubleSortedElements) Pop() Element {
	e := D.PrioritySorted[0]
	D.PrioritySorted = D.PrioritySorted[1:]
	delete(D.NameIndex, e.Name)
	return e
}

func (Q *Queue) AddElement(Name string, Function Command, Priority int) (SafeReturn, int) {
	Q.Lock.Lock(); Q.ExecLock.Lock()
	defer Q.Lock.Unlock(); defer Q.ExecLock.Unlock()
	for _, e := range(Q.ExecElements) { if Name == e.Name { return e.OutChannel, 0 } }
	e := Element{Name, Function, Priority, make(SafeReturn, 1)}
	return Q.Elements.AddElement(e)
}

func (Q *Queue) Exec(e Element) {
	defer func() {
		Q.ExecLock.Lock()
		defer Q.ExecLock.Unlock()
		for i, elem := range(Q.ExecElements) {
			if elem.Name == e.Name {
				Q.ExecElements = append(Q.ExecElements[:i], Q.ExecElements[i+1:]...)
				break
			}
		}
	}()
	defer func() {
		r := ""
		defer func() { e.OutChannel.Return(r) }()
		r = e.Function(e.Name)
	}()
	Q.ExecLock.Lock(); defer Q.ExecLock.Unlock()
	Q.ExecElements = append(Q.ExecElements, e)
}

func (Q *Queue) Init(Wait time.Duration, SimultaneousLimit int) {
	Q.Lock = new(sync.Mutex)
	Q.ExecLock = new(sync.Mutex)
	Q.Elements = DoubleSortedElements{make(map[string]int, 0), make([]Element, 0)}
	Q.ExecElements = make([]Element, 0)
	Q.Wait = Wait
	Q.SimultaneousLimit = SimultaneousLimit
}

func (Q *Queue) Run() {
	a := time.Tick(Q.Wait)
	for _ = range(a) {
		Q.Lock.Lock()
		if len(Q.Elements.PrioritySorted) > 0 && len(Q.ExecElements) < Q.SimultaneousLimit {
			e := Q.Elements.Pop()
			go Q.Exec(e)
		}
		Q.Lock.Unlock()
	}
}