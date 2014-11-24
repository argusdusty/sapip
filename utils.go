package sapip

type SafeReturn chan string

func (SR SafeReturn) Return(value string) { SR <- value }
func (SR SafeReturn) Read() string        { value := <-SR; SR <- value; return value }

type Element struct {
	Name       string
	Data       []string
	OutChannel SafeReturn
	Next       *Element
}

type PriorityElement struct {
	Name       string
	Data       []string
	Priority   int
	OutChannel SafeReturn
	Prev       *PriorityElement
	Next       *PriorityElement
}

type IndexedElements struct {
	NameIndex map[string]*Element // Map from each name to pointer to corresponding element
	Front     *Element            // Front element
	End       *Element            // Last element
}

var logFunc func(v ...interface{}) = log.Println

func setLogFunc(f func(v ...interface{})) {
	logFunc = f
}

func (D *IndexedElements) AddElement(Name, Data string) SafeReturn {
	if p, ok := D.NameIndex[Name]; ok {
		found := false
		for _, d := range p.Data {
			if d == Data {
				found = true
				break
			}
		}
		if !found {
			p.Data = append(p.Data, Data)
		}
		return p.OutChannel
	}
	e := &Element{Name, []string{Data}, make(SafeReturn, 1), nil}
	if D.Front == nil {
		D.Front = e
	} else {
		D.End.Next = e
	}
	D.End = e
	D.NameIndex[e.Name] = e
	return e.OutChannel
}

func (D *IndexedElements) Pop() *Element {
	e := D.Front
	D.Front = e.Next
	e.Next = nil
	delete(D.NameIndex, e.Name)
	return e
}

type IndexedPriorityElements struct {
	NameIndex      map[string]*PriorityElement // Map from each name to pointer to corresponding element
	PriorityMap    map[int]*PriorityElement    // Map from each priority to the element which is at the end of that priority
	Priorities     []int                       // List of priorities in sorted order
	PriorityLength map[int]int                 // Map from each priority to the number of elements which have the priority
	Front          *PriorityElement            // Front element
}

func (D *IndexedPriorityElements) AddPriority(Priority int) int {
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

func (D *IndexedPriorityElements) Remove(e *PriorityElement) {
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

func (D *IndexedPriorityElements) Add(e *PriorityElement) {
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

func (D *IndexedPriorityElements) AddElement(Name, Data string, Priority int) SafeReturn {
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
			e := &PriorityElement{Name, p.Data, Priority, p.OutChannel, nil, nil}
			D.Remove(p)
			D.Add(e)
		}
		return p.OutChannel
	}
	// Go ahead and insert the element
	e := &PriorityElement{Name, []string{Data}, Priority, make(SafeReturn, 1), nil, nil}
	D.Add(e)
	return e.OutChannel
}

func (D *IndexedPriorityElements) Pop() *PriorityElement {
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
