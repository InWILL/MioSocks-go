package engine

type Queue[T any] struct {
	queue []T
}

func (q *Queue[T]) front() T {
	return q.queue[0]
}

func (q *Queue[T]) push(t T) {
	q.queue = append(q.queue, t)
}

func (q *Queue[T]) pop() {
	q.queue = q.queue[1:]
}

func (q *Queue[T]) empty() bool {
	return len(q.queue) == 0
}
