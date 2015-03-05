SAPIP: Safe Asynchronous Periodic Indexed Priority Queue
========================================================
Safe: Multiple threads can access the return value of each command
Asynchronous: Multiple commands can be run simultaneously, up to a defined limit, and the queue can be accessed from multiple threads simultaneously. <br>
Periodic: The top command is executed/removed on a periodic basis, over a given delay.  <br>
Indexed: Each command is uniquely defined by a string, so that attempts at duplicating the command are ignored.  <br>
Priority: Lowest priority command goes first. Supports multiple commands having the same priority.  <br>
Queue: First in, first out, for each priority.  <br>

The queue function takes a string (their index/name) and a list a strings (set of commands to be run under the name) as input and returns a string. Feel free to modify those data types to support other return/input values. I didn't want to use the generalized interface{} for performance reasons.

Includes: <br>
SAPIPQueue - The full priority queue that runs commands at set intervals. <br>
SAIPQueue - A priority queue that runs commands as fast as possible. <br>
SAPIQueue - A periodic queue without priorities that runs commands at set intervals. <br>
SAIQueue - A bare-bones queue without priorities that runs commands as fast as possible. <br>

***Author:*** Mark Canning <br>
***Email:*** argusdusty@gmail.com <br>
***Developed at/for:*** [Tamber](http://tamber.com/)

Installing
----------
Install: `go get github.com/argusdusty/sapip` <br>
Update: `go get -u github.com/argusdusty/sapip` <br>
Using: `import "github.com/argusdusty/sapip"` <br>

Example usage
-----------------------

You have an API that restricts you to a certain rate limit (2 API calls/second, for example), and certain API calls should be run at a higher priority (for example, user requests > background calls):

```go
var APIQueue = sapip.NewSAPIPQueue(HandleAPICall, 8)

func init() {
	// You may want to add a small buffer onto the 500ms
	go APIQueue.Run(500*time.Millisecond)
}

// Ignore data
func HandleAPICall(command string, data []string) string {
	return execAPICall(command)
}

// Lowest priority runs first
func APICall(command string, priority int) string {
	reader := APIQueue.AddElement(command, "", priority)
	return reader.Read()
}
```

You have a data structure that must be loaded from the DB and analyzed in order to be modified, and there are several possible modifications that can be made. You want this to run as fast as possible, including running multiple modifications to the same object together so that you don't have to reload the object each time, but you want to avoid having the same object being accessed simultaneously.

```go
var ObjectQueue = sapip.NewSAIQueue(HandleModifyObject, 8)

func init() {
	go ObjectQueue.Run()
}

func HandleModifyObject(objectKey string, modifications []string) string {
	object := loadObject(objectKey)
	for _, modification := range modifications {
		RunModification(object, modification)
	}
	saveObject(object, objectKey)
	return ""
}

func ModifyObject(objectKey string, modification string, waitFinish bool) {
	reader := ObjectQueue.AddElement(objectKey, modification)
	// We can choose to not wait for the changes to complete, and let them run in the background
	if waitFinish {
		reader.Read()
	}
}
```

Copyright (C) 2015  Mark Canning