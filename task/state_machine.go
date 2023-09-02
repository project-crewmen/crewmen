package task

/*
Define the state of a task's life cycle.
-- Initial State is Pending.
-- When its scheduled to be run, then its in the Scheduled state. (Manager schedules a task onto a worker)
-- If the machine successfully started the task, then its in the Running state. (Worker run the task)
-- Once task executed successfully, its in Completed state.
-- If task crashes at some point then its Failed state.
*/
type State int

const (
	Pending State = iota
	Scheduled
	Running
	Completed
	Failed
)

// State Machines transition relations - Possible state transitions
var stateTransitionMap = map[State][]State{
	Pending:   []State{Scheduled},
	Scheduled: []State{Scheduled, Running, Failed},
	Running:   []State{Running, Completed, Failed},
	Completed: []State{},
	Failed:    []State{},
}

// Check whether an specific state is in a slice of states
func Contains(states []State, state State) bool {
	for _, s := range states {
		if s == state {
			return true
		}
	}

	return false
}

// Helper function - Check whether a task's state can be transition from given source to destination state
func ValidStateTransition(src State, dst State) bool {
	return Contains(stateTransitionMap[src], dst)
}
