package sapip

import (
	"log"
	"sync"
	"time"
)

type SafeReturn chan string

func (SR SafeReturn) Return(value string) { SR <- value }
func (SR SafeReturn) Read() string        { value := <-SR; SR <- value; return value }
func (SR SafeReturn) Close()              {}

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
	PriorityLength map[int]int         // Map from each priority to the number of elements which have the priority
	Front          *Element            // Front element
}

type Queue struct {
	Lock              *sync.Mutex // Global lock
	ExecLock          *sync.Mutex // Lock for manipulating ExecElements
	EmptyCond         *sync.Cond  // Wait for queue to be non-empty
	LimitCond         *sync.Cond  // Wait for open slot in ExecElements
	Elements          DoubleSortedElements
	ExecElements      []*Element
	SimultaneousLimit int
	Function          func(name string, data []string) string
	Stopped           bool
}

func (Q *Queue) Init(SimultaneousLimit int, Function func(name string, data []string) string) {
	Q.Lock = new(sync.Mutex)
	Q.ExecLock = new(sync.Mutex)
	Q.EmptyCond = sync.NewCond(new(sync.Mutex))
	Q.LimitCond = sync.NewCond(new(sync.Mutex))
	Q.Elements = DoubleSortedElements{make(map[string]*Element, 0), make(map[int]*Element, 0), make([]int, 0), make(map[int]int, 0), nil}
	Q.ExecElements = make([]*Element, 0)
	Q.SimultaneousLimit = SimultaneousLimit
	Q.Function = Function
	Q.Stopped = false
}

func (D *DoubleSortedElements) AddPriority(Priority int) int {
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

func (D *DoubleSortedElements) Remove(e *Element) {
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
		// Note to self: This might just be faster as a linear search. How many distinct priorities could there be?
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

func (D *DoubleSortedElements) Add(e *Element) {
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
		i := D.AddPriority(e.Priority)
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

func (D *DoubleSortedElements) AddElement(Name, Data string, Priority int) SafeReturn {
	// If the element name is already in the queue we need to do special stuff
	if p, ok := D.NameIndex[Name]; ok {
		// Check if the data we're trying to insert is already there
		found := false
		for _, d := range p.Data {
			if d == Data {
				found = true
				break
			}
		}
		// If not, add it, otherwise do nothing
		if !found {
			p.Data = append(p.Data, Data)
		}
		// If the new priority is smaller, we need to remove the old element and insert it with the new priority
		if p.Priority > Priority {
			// Note to self: Possibility of memory leak in re-using p.Data/p.OutChannel?
			e := &Element{Name, p.Data, Priority, p.OutChannel, nil, nil}
			D.Remove(p)
			D.Add(e)
		}
		return p.OutChannel
	}
	// Go ahead and insert the element
	e := &Element{Name, []string{Data}, Priority, make(SafeReturn, 1), nil, nil}
	D.Add(e)
	return e.OutChannel
}

func (D *DoubleSortedElements) Pop() *Element {
	e := D.Front
	// Set the front to the next element and clear its pointer to e
	D.Front = e.Next
	if e.Next != nil {
		e.Next.Prev = nil
	}
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

func (Q *Queue) Exec(e *Element) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Error in queue on element:", e.Name, "-", r)
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
		Q.EmptyCond.L.Lock()
		defer Q.EmptyCond.L.Unlock()
		Q.EmptyCond.Broadcast()
	}()
	// Execute the function and return it in a defer (in case it panics)
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
		// Wait for the queue to be non-empty
		Q.EmptyCond.L.Lock()
		for len(Q.Elements.Priorities) == 0 {
			Q.EmptyCond.Wait()
		}
		Q.EmptyCond.L.Unlock()
		// Wait for a free slot in ExecElements
		Q.LimitCond.L.Lock()
		for len(Q.ExecElements) >= Q.SimultaneousLimit {
			Q.LimitCond.Wait()
		}
		Q.LimitCond.L.Unlock()
		// Put these in func()s to make sure the locks return (in case of some weird unexpected panic)
		func() {
			Q.Lock.Lock()
			defer Q.Lock.Unlock()
			e := Q.Elements.Pop()
			go func() {
				Q.ExecLock.Lock()
				defer Q.ExecLock.Unlock()
				// Don't run until current tasks with same name are finished
				Q.ExecElements = append(Q.ExecElements, e)
				for _, elem := range Q.ExecElements {
					if elem.Name == e.Name && elem != e {
						elem.OutChannel.Read()
					}
				}
				go Q.Exec(e)
			}()
		}()
	}
}
