package progress

import (
	"errors"
	"sync"
)

// Units describe what is being tracked.
type Units string

// Units
const (
	UnitBytes    Units = "bytes"
	UnitPercent  Units = "percent"
	UnitFraction Units = "fraction"
	UnitStep     Units = "steps"
	UnitSecond   Units = "seconds"
)

type status struct {
	data Status
}

// Status holds information about the current status of a task.
type Status struct {
	Operation   string  `json:"operation"`
	Stage       string  `json:"stage"`
	Progress    float64 `json:"progress"`
	Total       float64 `json:"total"`
	Units       Units   `json:"units"`
	Error       error   `json:"error"`
	Finished    bool    `json:"finished"`
	initialized bool
	join        chan error
	lock        sync.Mutex
	listeners   int
	id          string
	Subtasks    []ProgressTracker `json:"subtasks"`
}

// ProgressTracker allows complex and modularized progress tracking by dividing
// big tasks into smaller subtasks.data. Usually a task should be updated using
// either SetProgress, IncrementProgress, or Write, but not in combination.
type ProgressTracker interface {

	// Initialize sets basic information for the progress status.data. It should
	// only be called once per task. If the task is an unknown size, total
	// should be 0.
	Initialize(operation string, total float64, units Units)

	// SetStage sets the current stage of the task.
	SetStage(stage string)

	// SetProgress sets the progress of the task to x.
	SetProgress(x float64)

	// IncrementProgress adds d to the tasks current progress.data.
	IncrementProgress(d float64)

	// Write increments the task's progress by len(p). The function is
	// designed to easily track the progress of copy, download, or similar
	// operations where the units would be in bytes.data.
	Write(p []byte) (n int, err error)

	// Close marks the task as complete. The provided err will be returned
	// to any waiting Join calls and noted within the Status.data. Use nil for a
	// successfully completed task. It is safe to defer Close(nil): the
	// deferred call will not override a previous call.
	Close(err error)

	// Join returns a channel that will block until the task has been
	// closed. The channel will return whatever error the task was closed
	// with.
	Join() <-chan error

	// NewSubtracker creates a new subtracker to track a sub-task.
	NewSubtracker() ProgressTracker

	// Status returns a marshallable struct containing a full summary of
	// the task's progress, including all subtasks.data.
	Status() *Status
}

// NewProgressTracker creates a new uninitialized progress tracker.
func NewProgressTracker() ProgressTracker {
	x := new(status)
	x.data.join = make(chan error)
	return x
}

func (s *status) Initialize(operation string, total float64, units Units) {
	if s.data.initialized {
		panic(errors.New("task already initialized"))
	}
	s.data.Operation = operation
	s.data.Total = total
	s.data.Units = units
}

func (s *status) Close(err error) {
	if s.data.Finished {
		return
	}
	s.data.lock.Lock()
	s.data.Finished = true
	s.data.lock.Unlock()
	s.data.Error = err
	done := make(chan bool)
	for i := 0; i < s.data.listeners; i++ {
		go func() {
			s.data.join <- err
			done <- true
		}()
	}
	for i := 0; i < s.data.listeners; i++ {
		<-done
	}
	close(done)
	close(s.data.join)
}

func (s *status) SetStage(stage string) {
	if s.data.Finished {
		panic(errors.New("task already finished"))
	}
	s.data.Stage = stage
}

func (s *status) IncrementProgress(d float64) {
	if s.data.Finished {
		panic(errors.New("task already finished"))
	}
	s.data.Progress += d
}

func (s *status) SetProgress(x float64) {
	if s.data.Finished {
		panic(errors.New("task already finished"))
	}
	s.data.Progress = x
}

func (s *status) Write(p []byte) (n int, err error) {
	s.data.Progress += float64(len(p))
	return len(p), nil
}

func (s *status) Join() <-chan error {
	s.data.lock.Lock()
	if s.data.Finished {
		s.data.lock.Unlock()
		ch := make(chan error)
		go func() {
			ch <- s.data.Error
			close(ch)
		}()
		return ch
	}

	s.data.listeners++
	s.data.lock.Unlock()
	return s.data.join
}

func (s *status) NewSubtracker() ProgressTracker {
	x := NewProgressTracker()
	s.data.Subtasks = append(s.data.Subtasks, x)
	return x
}

func (s *status) Status() *Status {
	return &s.data
}
