package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"../labrpc"
)

// import "bytes"
// import "../labgob"

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

// LogEntry contains a command for the state machine, and the term when the
// entry was received by the leader.
type LogEntry struct {
	Command interface{}
	Term    int
}

type State int

const (
	Follower State = iota
	Candidate
	Leader
)

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	currentTerm   int         // latest term server has seen
	votedFor      int         // candidateId that received vote in current term (or null if none)
	state         State       // server state: Follower, Candidate, Leader
	log           []LogEntry  // log entries
	// Other state
	electionTimer *time.Timer // election timer
	// Volatile state on all servers:
	commitedIndex  int         // index of highest log entry known to be committed
	lastAppliedIndex int       // index of highest log entry applied to state machine
	nextIndex      []int       // for each server, index of the next log entry to send to that server
	matchIndex     []int       // for each server, index of highest log entry known to be replicated on server
	applyCh        chan ApplyMsg // channel to send ApplyMsg to service (or tester)
	applyCond     *sync.Cond // Used to signal the applier goroutine
}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	CandidateId  int // candidate requesting vote
	Term         int // candidate's term
	LastLogIndex int // index of candidate's last log entry
	LastLogTerm  int // term of candidate's last log entry
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	Term        int  // currentTerm, for candidate to update itself
	VoteGranted bool // true means candidate received vote
}

type AppendEntriesArgs struct {
	Term         int        // leader's term
	LeaderId     int        // so follower can redirect clients
	PrevLogIndex int        // index of log entry immediately preceding new ones
	PrevLogTerm  int        // term of prevLogIndex entry
	Entries      []LogEntry // log entries to store (empty for heartbeat; may send more than one for efficiency)
	LeaderCommit int        // leader's commitIndex
}

type AppendEntriesReply struct {
	Term    int  // currentTerm, for leader to update itself
	Success bool // true if follower contained entry matching prevLogIndex and prevLogTerm
	// For 2B
	ConflictIndex int // index of first entry with term equal to ConflictTerm
	ConflictTerm  int // term of the conflicting entry
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	var term int
	var isleader bool

	rf.mu.Lock()
	defer rf.mu.Unlock()
	term = rf.currentTerm
	isleader = (rf.state == Leader)
	return term, isleader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
func (rf *Raft) persist() {	
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// startElection is called to start a new election.
func (rf *Raft) startElection() {
	rf.currentTerm++
	rf.state = Candidate
	rf.votedFor = rf.me
	rf.electionTimer.Reset(randomizedElectionTimeout())

	args := &RequestVoteArgs{
		Term:        rf.currentTerm,
		CandidateId: rf.me,
		LastLogIndex: len(rf.log) - 1,
		LastLogTerm:  rf.log[len(rf.log)-1].Term,
	}

	votesReceived := 1 // Vote for self

	for i := range rf.peers {
		if i == rf.me {
			continue
		}

		go func(serverIndex int) {
			reply := &RequestVoteReply{}
			ok := rf.sendRequestVote(serverIndex, args, reply)

			if ok {
				rf.mu.Lock()
				defer rf.mu.Unlock()

				if rf.state != Candidate || rf.currentTerm != args.Term {
					return
				}

				if reply.Term > rf.currentTerm {
					rf.state = Follower
					rf.currentTerm = reply.Term
					rf.votedFor = -1
					return
				}

				if reply.VoteGranted {
					votesReceived++
					if votesReceived > len(rf.peers)/2 {
						if rf.state == Candidate {
							rf.becomeLeader()
							rf.broadcastAppendEntries() // Send initial empty AppendEntries RPCs (heartbeats) to each server
						}
					}
				}
			}
		}(i)
	}
}


// Must be called with lock held.
func (rf *Raft) becomeLeader() {
    if rf.state != Candidate {
        return
    }
    rf.state = Leader
    lastLogIndex := len(rf.log) - 1
	lastLogTerm := rf.log[lastLogIndex].Term
	DPrintf("Server %d became leader for term %d at log index %d, term %d", rf.me, rf.currentTerm, lastLogIndex, lastLogTerm)
    rf.nextIndex = make([]int, len(rf.peers))
    rf.matchIndex = make([]int, len(rf.peers))
    for i := range rf.peers {
        rf.nextIndex[i] = lastLogIndex + 1
        rf.matchIndex[i] = 0
    }
}

// Must be called with lock held.
func (rf *Raft) becomeFollower(term int) {
    rf.state = Follower
    rf.currentTerm = term
    rf.votedFor = -1
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).

	rf.mu.Lock()
	defer rf.mu.Unlock()

	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}

	if args.Term > rf.currentTerm {
		rf.becomeFollower(args.Term)
	}

	canVote := rf.votedFor == -1 || rf.votedFor == args.CandidateId

	myLastLogIndex := len(rf.log) - 1
	myLastLogTerm := rf.log[myLastLogIndex].Term
	isLogUpToDate := (args.LastLogTerm > myLastLogTerm) || (args.LastLogTerm == myLastLogTerm && args.LastLogIndex >= myLastLogIndex)

	if canVote && isLogUpToDate {
		rf.votedFor = args.CandidateId
		reply.VoteGranted = true
		rf.electionTimer.Reset(randomizedElectionTimeout())
	} else {
		reply.VoteGranted = false
	}

}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}

	rf.electionTimer.Reset(randomizedElectionTimeout())
	rf.becomeFollower(args.Term)

	reply.Term = rf.currentTerm

	if args.PrevLogIndex > 0 { // The dummy entry at index 0 always matches
		lastLogIndex := len(rf.log) - 1
        if args.PrevLogIndex > lastLogIndex {
            // Follower's log is too short
            reply.Success = false
            reply.ConflictIndex = lastLogIndex + 1
            reply.ConflictTerm = -1 // No conflicting term
            return
        }
        if rf.log[args.PrevLogIndex].Term != args.PrevLogTerm {
            // Terms do not match
            reply.Success = false
            reply.ConflictTerm = rf.log[args.PrevLogIndex].Term
            // Find the first index of this conflicting term
			firstIndexOfTerm := 1
            for i := args.PrevLogIndex; i > 0; i-- {
                if rf.log[i-1].Term != reply.ConflictTerm {
                    firstIndexOfTerm = i
					break
                }
            }
			reply.ConflictIndex = firstIndexOfTerm
            return
        }
    }

	for i, entry := range args.Entries {
        logIndex := args.PrevLogIndex + 1 + i
        if logIndex > len(rf.log)-1 {
            rf.log = append(rf.log, args.Entries[i:]...)
            break
        }
        if rf.log[logIndex].Term != entry.Term {
            rf.log = rf.log[:logIndex]
            rf.log = append(rf.log, args.Entries[i:]...)
            break
        }
    }

    reply.Success = true

    // Update commit index
    if args.LeaderCommit > rf.commitedIndex {
        lastNewEntryIndex := args.PrevLogIndex + len(args.Entries)
        rf.commitedIndex = min(args.LeaderCommit, lastNewEntryIndex)
        // rf.applyCond.Signal()
    }
}


// sendRequestVote is already provided, but you call it like this.
// It's a wrapper around rf.peers[server].Call("Raft.RequestVote", args, reply)
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.state != Leader {
		isLeader = false
		return index, term, isLeader
	}

	newEntry := LogEntry{
		Term:    rf.currentTerm,
		Command: command,
	}

	rf.log = append(rf.log, newEntry)

	index = len(rf.log) - 1
	term = rf.currentTerm

	rf.broadcastAppendEntries()

	return index, term, isLeader
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// Helper function to get a randomized election timeout
func randomizedElectionTimeout() time.Duration {
	return time.Duration(300+rand.Intn(300)) * time.Millisecond
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
    ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
    return ok
}

// Must be called with lock held.
func (rf *Raft) updateCommitIndex() {
    // If there exists an N such that N > commitIndex, a majority
    // of matchIndex[i] ≥ N, and log[N].term == currentTerm:
    // set commitIndex = N
	lastLogIndex := len(rf.log) - 1
    for N := lastLogIndex; N > rf.commitedIndex; N-- {
        // Only commit entries from the current term
        if rf.log[N].Term == rf.currentTerm {
            count := 1 // Count self
            for i := range rf.peers {
                if i != rf.me && rf.matchIndex[i] >= N {
                    count++
                }
            }
            if count > len(rf.peers)/2 {
                rf.commitedIndex = N
                // rf.applyCond.Signal() // Wake up applier
                break
            }
        }
    }
}

func (rf *Raft) replicateLogToPeer(server int) {
    rf.mu.Lock()
    if rf.state != Leader {
        rf.mu.Unlock()
        return
    }

    prevLogIndex := rf.nextIndex[server] - 1
    if prevLogIndex < 0 {
        // This should not happen with a dummy entry at index 0.
        prevLogIndex = 0
    }
    
    prevLogTerm := 0
    if prevLogIndex > 0 {
        prevLogTerm = rf.log[prevLogIndex].Term
    }

    entries := make([]LogEntry, len(rf.log[prevLogIndex+1:]))
    copy(entries, rf.log[prevLogIndex+1:])

    args := AppendEntriesArgs{
        Term:         rf.currentTerm,
        LeaderId:     rf.me,
        PrevLogIndex: prevLogIndex,
        PrevLogTerm:  prevLogTerm,
        Entries:      entries,
        LeaderCommit: rf.commitedIndex,
    }
    rf.mu.Unlock()

    reply := AppendEntriesReply{}
	ok := rf.sendAppendEntries(server, &args, &reply)
	//
    if ok {
        rf.mu.Lock()
        defer rf.mu.Unlock()

        if rf.state != Leader || rf.currentTerm != args.Term {
            return
        }

        if reply.Term > rf.currentTerm {
            rf.becomeFollower(reply.Term)
            return
        }

        if reply.Success {
            rf.matchIndex[server] = args.PrevLogIndex + len(args.Entries)
            rf.nextIndex[server] = rf.matchIndex[server] + 1
            rf.updateCommitIndex()
        } else {
            // Log inconsistency. Decrement nextIndex and retry.
            // Use the conflict information for faster reconciliation.
            if reply.ConflictTerm != -1 {
                // Find the last index of ConflictTerm in our log
                lastConflictTermIndex := -1
                for i := len(rf.log) - 1; i > 0; i-- {
                    if rf.log[i].Term == reply.ConflictTerm {
                        lastConflictTermIndex = i
                        break
                    }
                }
                if lastConflictTermIndex != -1 {
                    rf.nextIndex[server] = lastConflictTermIndex + 1
                } else {
                    rf.nextIndex[server] = reply.ConflictIndex
                }
            } else {
                rf.nextIndex[server] = reply.ConflictIndex
            }
            // nextIndex doesn't go below 1
            if rf.nextIndex[server] < 1 {
                rf.nextIndex[server] = 1
            }
        }
    }
}

func (rf *Raft) broadcastAppendEntries() {
    for i := range rf.peers {
        if i == rf.me {
            continue
        }
        go rf.replicateLogToPeer(i)
    }
}

// The ticker go routine starts a new election if this peer hasn't received heartbeats recently.
func (rf *Raft) ticker() {
	for !rf.killed() {
		time.Sleep(10 * time.Millisecond)

		rf.mu.Lock()
        state := rf.state
        rf.mu.Unlock()

		if state == Leader {
			rf.mu.Lock()
			rf.broadcastAppendEntries()
			rf.mu.Unlock()
            time.Sleep(100 * time.Millisecond) // Heartbeat interval
		} else {
			select {
			case <-rf.electionTimer.C:
				rf.mu.Lock()
				rf.startElection()
				rf.mu.Unlock()
			default:
				
			}
		}
	}
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.state = Follower
	rf.votedFor = -1
	rf.currentTerm = 0
	rf.log = make([]LogEntry, 1)
	rf.log[0] = LogEntry{Term: 0, Command: nil}
	
	rf.commitedIndex = 0
    rf.lastAppliedIndex = 0
	// rf.nextIndex = make([]int, len(rf.peers))
	// rf.matchIndex = make([]int, len(rf.peers))
    rf.applyCh = applyCh
	rf.applyCond = sync.NewCond(&rf.mu)

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	rf.electionTimer = time.NewTimer(randomizedElectionTimeout())

	go rf.ticker()

	return rf
}
