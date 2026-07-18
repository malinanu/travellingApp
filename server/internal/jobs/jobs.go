// Package jobs runs fire-and-forget work (email sends today; any other
// slow side effect tomorrow) on a small background worker pool, so request
// handlers that trigger it don't block on it — e.g. creating a team invite
// or requesting a password reset returns immediately, and the email goes
// out shortly after on its own goroutine.
package jobs

import "log"

// Queue is a bounded, best-effort task queue. It is not a durable job
// system: queued work that hasn't started yet is lost on process restart,
// and a full queue drops new tasks rather than blocking the caller. That's
// an acceptable tradeoff for "send this email" (the invite/reset code
// still works via its own copy-link/retry, so a dropped email isn't
// catastrophic) — it would not be for anything that must eventually run.
type Queue struct {
	ch chan func()
}

// NewQueue starts workers goroutines pulling from a queue of size
// backlog, and returns immediately — callers don't wait for workers to
// start.
func NewQueue(workers, backlog int) *Queue {
	q := &Queue{ch: make(chan func(), backlog)}
	for i := 0; i < workers; i++ {
		go q.run()
	}
	return q
}

func (q *Queue) run() {
	for fn := range q.ch {
		runSafely(fn)
	}
}

// runSafely isolates one task's panic from the worker goroutine itself —
// a bad task shouldn't take down the whole background pool.
func runSafely(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("jobs: task panicked: %v", r)
		}
	}()
	fn()
}

// Enqueue schedules fn to run asynchronously. If the queue is full, fn is
// dropped (logged) rather than blocking the caller — queued work is
// best-effort, not guaranteed.
func (q *Queue) Enqueue(fn func()) {
	select {
	case q.ch <- fn:
	default:
		log.Println("jobs: queue full, dropping task")
	}
}
