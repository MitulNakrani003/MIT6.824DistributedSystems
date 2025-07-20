package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import "os"
import "strconv"

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.
type TaskType int
const (
	MapTask	TaskType = iota
	ReduceTask
	WaitTask
	ExitTask
)

type TaskRequestArgs struct {
    WorkerID int
}

type TaskRequestReply struct {
    TaskType   TaskType
    MapID     int
    Filename   string   // For map tasks
    NReduce    int      // For map tasks
    ReduceID   int      // For reduce tasks
}

type TaskReportArgs struct {
    TaskType   TaskType
    TaskID     int
    Success    bool
}

type TaskReportReply struct {
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the master.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func masterSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
