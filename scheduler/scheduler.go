package scheduler

// Assign a task to a worker

type Scheduler interface {
	SelectCandidateNodes()
	Score()
	Pick()
}