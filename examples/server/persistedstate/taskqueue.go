package persistedstate

import "sync"

// TaskQueue mirrors TaskQueue.ts: tasks run one at a time in the order they were
// added. AddTask called while the queue is draining only enqueues; the draining
// loop runs the task afterwards. Tasks run on the goroutine that adds the first
// task of a drain (JS awaits them on the event loop).
type TaskQueue struct {
	mu         sync.Mutex
	tasks      []func()
	processing bool
}

// AddTask mirrors taskQueue.addTask(task).
func (q *TaskQueue) AddTask(task func()) {
	q.mu.Lock()
	q.tasks = append(q.tasks, task)
	if q.processing {
		q.mu.Unlock()
		return
	}
	q.processing = true
	defer func() {
		q.processing = false
		q.mu.Unlock()
	}()
	for len(q.tasks) > 0 {
		next := q.tasks[0]
		q.tasks = q.tasks[1:]
		q.mu.Unlock()
		next()
		q.mu.Lock()
	}
}
