# 🧠 Distributed Systems Labs – MIT 6.824
### *"From Parallel Processing to Consensus – Building the Foundations of Distributed Computing."*

---

## 🚀 Overview
This repository contains implementations of all the Labs from MIT's **6.824: Distributed Systems** course.
Each lab progressively deepens understanding of building **fault-tolerant, parallel, and replicated systems** using the Go programming language.

---

## 🧩 Lab 1 – MapReduce
**Goal:** Build a simplified distributed MapReduce system that runs user-defined map and reduce tasks in parallel.

### ✳️ Highlights
* Implemented **Master–Worker coordination** via **Go RPC**, handling dynamic task allocation and worker crashes.
* Supported **fault recovery** by re-assigning tasks after timeout detection.
* Generated intermediate files using **JSON encoding** for deterministic reduce-phase aggregation.
* Achieved **100% pass rate** on the parallelism, crash recovery, and correctness tests.

### 💡 Key Learnings
* Designing **distributed task scheduling** under failure conditions.
* Managing concurrency with **Go goroutines and synchronization primitives**.
* Applying atomic file operations (`os.Rename`) to ensure crash-safe writes.
* Gaining deep insight into the **MapReduce paper** through practical re-implementation.

---

## 🔁 Lab 2 – Raft Consensus Algorithm
**Goal:** Implement the **Raft** consensus protocol to maintain replicated logs and ensure consistent state across unreliable networks.

### ✳️ Highlights
* Built a **leader election**, **log replication**, and **persistence** mechanism across simulated servers.
* Implemented all three parts of the lab:
  * **2A:** Leader election and heartbeat mechanism.
  * **2B:** Log replication and follower consistency.
  * **2C:** State persistence and recovery after crash or reboot.
* Verified correctness with 100% passing scores on all test suites (2A, 2B, 2C).
* Optimized election timeouts and RPC scheduling for deterministic recovery and efficient consensus.

### 💡 Key Learnings
* Developed an in-depth understanding of **distributed consensus** and **fault tolerance**.
* Learned how to maintain **replicated state machines** that remain consistent under partial failure.
* Practiced **lock management, concurrency control**, and **Go RPC message flow debugging**.
* Experienced real-world reliability engineering: heartbeat intervals, election backoffs, and log compaction design trade-offs.

---

## 🗄️ Lab 3 – Fault-tolerant Key/Value Service
**Goal:** Build a **linearizable, fault-tolerant key/value storage service** using Raft for replication, providing strong consistency guarantees.

### ✳️ Highlights
* Implemented a **replicated state machine** architecture with KVServers backed by Raft consensus.
* Built two major components:
  * **3A:** Key/value service with linearizability and exactly-once semantics
  * **3B:** Log compaction via snapshotting to prevent unbounded memory growth
* Key features implemented:
  * **Client request deduplication** using ClientID and sequence numbers for idempotency
  * **Notification channels** for efficient waiting on Raft commit confirmations
  * **Leader detection and retry logic** with smart leader caching
  * **Snapshot installation** with InstallSnapshot RPC for catching up lagging followers
  * **Conditional snapshot installation** (`CondInstallSnapshot`) to prevent stale snapshot overwrites

### 🔧 Technical Details
* **Linearizability:** All operations (Get/Put/Append) appear to execute atomically at some point between their invocation and response
* **Exactly-once semantics:** Handled duplicate client requests through sequence number tracking
* **Memory management:** Implemented log compaction when Raft state approaches `maxraftstate` threshold
* **State persistence:** Snapshot includes both key-value database and deduplication state
* **Fault tolerance:** Service continues operating as long as a majority of servers are available

### 💡 Key Learnings
* Mastered **building applications on top of consensus protocols** (Raft as a black box)
* Implemented **linearizable distributed storage** with strong consistency guarantees
* Designed **efficient client-server interaction patterns** for retry and leader discovery
* Learned **snapshot-based log compaction** strategies for long-running services
* Practiced **cross-layer coordination** between application (KVServer) and consensus (Raft) layers
* Understood the critical importance of **idempotency** in distributed systems
* Gained experience with **state machine replication** and **deterministic execution**

### 📊 Performance Metrics
* Passed all Lab 3A tests (15 tests) including:
  * Linearizability checks under various failure scenarios
  * Network partitions and healing
  * Server restarts and unreliable networks
* Passed all Lab 3B tests (8 tests) including:
  * InstallSnapshot RPC functionality
  * Reasonable snapshot sizes
  * Recovery from snapshots after restarts
* Typical test completion: ~400 seconds real time, ~700 seconds CPU time

---

## 🛠️ Tech Stack
* **Language:** Go (1.13+)
* **Concurrency:** goroutines, channels, mutexes, `sync.Cond`
* **Persistence:** Custom in-memory persister abstraction with snapshot support
* **RPC Framework:** Go net/rpc
* **Encoding:** GOB encoding for state serialization
* **Testing:** Comprehensive test suites including linearizability checkers
* **Architecture:** Layered design (Client → KVServer → Raft → Network)

---

## 📈 Impact & Applications
* Built **production-grade distributed systems** patterns from scratch
* Achieved **robust fault-tolerant computation and storage** with proven correctness
* Developed practical understanding of:
  * **CAP theorem** trade-offs in distributed systems
  * **Consensus-based replication** for high availability
  * **State machine replication** for deterministic distributed computation
  * **Log-structured storage** and compaction strategies
* Foundation for real-world systems like:
  * **Distributed databases** (CockroachDB, TiDB)
  * **Coordination services** (Zookeeper, etcd, Consul)
  * **Replicated state stores** in microservices architectures

---

## 🔍 Key Challenges Overcome
* **Race conditions:** Careful mutex management across concurrent RPC handlers and background goroutines
* **Deadlock prevention:** Structured locking hierarchy between KVServer and Raft layers
* **Network partitions:** Robust handling of split-brain scenarios and leader changes
* **Memory efficiency:** Balancing log retention with snapshot frequency
* **Duplicate detection:** Maintaining deduplication state across crashes and snapshots
* **Stale data prevention:** Ensuring followers never install outdated snapshots

---

## 📚 References
* [MapReduce Paper](https://pdos.csail.mit.edu/6.824/papers/mapreduce.pdf)
* [Extended Raft Paper](https://pdos.csail.mit.edu/6.824/papers/raft-extended.pdf)
* [MIT 6.824 Course Materials](https://pdos.csail.mit.edu/6.824/)


## 📝 License
This project is for educational purposes as part of MIT's 6.824 course.
