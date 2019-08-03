package laborunion

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

func New() *WorkerPool {
	return &WorkerPool{
		quitChan:             make(chan bool),
		inChan:               make(chan interface{}),
		retries:              3, // default
		maxRetryMilliseconds: 5, // default
		batchSize:            1, // default
	}
}

type WorkerPool struct {
	quitChan   chan bool
	inChan     chan interface{}
	inChanSize int

	// hooks
	beforeBatchingHook func()
	afterBatchingHook  func([]interface{})
	onFailedWork       func(error)
	onSuccessWork      func([]interface{})
	onNewWorker        func()
	onDeleteWorker     func()

	worker      func([]interface{}) error
	workerCount int

	retries              int
	maxRetryMilliseconds int
	batchSize            int

	mtx sync.RWMutex
}

// ------------------------------------
// Hooks
// ------------------------------------

// SetBeforeBatchingHook sets function to be executed before batching.
func (p *WorkerPool) SetBeforeBatchingHook(f func()) {
	p.mtx.Lock()
	p.beforeBatchingHook = f
	p.mtx.Unlock()
}

// GetBeforeBatchingHook returns function to be executed before batching.
func (p *WorkerPool) GetBeforeBatchingHook() func() {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.beforeBatchingHook
}

// SetAfterBatchingHook sets function to be executed after batching.
func (p *WorkerPool) SetAfterBatchingHook(f func([]interface{})) {
	p.mtx.Lock()
	p.afterBatchingHook = f
	p.mtx.Unlock()
}

// GetAfterBatchingHook returns function to be executed after batching.
func (p *WorkerPool) GetAfterBatchingHook() func([]interface{}) {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.afterBatchingHook
}

// SetOnNewWorker returns function to be executed when a worker is created.
func (p *WorkerPool) SetOnNewWorker(f func()) {
	p.mtx.Lock()
	p.onNewWorker = f
	p.mtx.Unlock()
}

// GetOnNewWorker returns function to be executed when a worker is created.
func (p *WorkerPool) GetOnNewWorker() func() {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.onNewWorker
}

// SetOnDeleteWorker returns function to be executed when a worker is deleted.
func (p *WorkerPool) SetOnDeleteWorker(f func()) {
	p.mtx.Lock()
	p.onDeleteWorker = f
	p.mtx.Unlock()
}

// GetOnDeleteWorker returns function to be executed when a worker is deleted.
func (p *WorkerPool) GetOnDeleteWorker() func() {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.onDeleteWorker
}

// SetOnFailedWork sets function to be executed when worker failed to finish.
func (p *WorkerPool) SetOnFailedWork(f func(error)) {
	p.mtx.Lock()
	p.onFailedWork = f
	p.mtx.Unlock()
}

// GetOnFailedWork returns function to be executed when worker failed to finish.
func (p *WorkerPool) GetOnFailedWork() func(error) {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.onFailedWork
}

// SetOnSuccessWork sets function to be executed when worker finished successfully.
func (p *WorkerPool) SetOnSuccessWork(f func([]interface{})) {
	p.mtx.Lock()
	p.onSuccessWork = f
	p.mtx.Unlock()
}

// GetOnSuccessWork returns function to be executed when worker finished successfully.
func (p *WorkerPool) GetOnSuccessWork() func([]interface{}) {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.onSuccessWork
}

// Run pulls tasks and use worker function to process the task.
// It must be executed as goroutine because it blocks forever.
func (p *WorkerPool) Run() error {
	if p.GetWorker() == nil {
		return fmt.Errorf("WorkerPool is missing the worker function")
	}

	for {
		// 1. Execute custom logic before batching happens.
		if p.GetBeforeBatchingHook() != nil {
			p.GetBeforeBatchingHook()()
		}

		tasks := make([]interface{}, 0)

		// 2. Pull N number of tasks from InChan and batch them into a slice.
		for i := 0; i < p.GetBatchSize(); i++ {
			select {
			case task, ok := <-p.GetInChan():
				if ok {
					tasks = append(tasks, task)
				}

			case <-p.GetQuitChan():
				return nil
			}
		}

		// 3. Execute custom logic after batching happens.
		if p.GetAfterBatchingHook() != nil {
			p.GetAfterBatchingHook()(tasks)
		}

		// 4. Loop through every single task and executes them.
		var err error

		if p.GetRetries() > 0 {
			for i := 0; i < p.GetRetries(); i++ {
				err = p.GetWorker()(tasks)
				if err == nil {
					break
				}

				// Sleep between retrying
				time.Sleep(p.RetryDuration())
			}

		} else {
			err = p.GetWorker()(tasks)
		}

		if err != nil {
			if p.GetOnFailedWork() != nil {
				p.GetOnFailedWork()(err)
			}
		} else {
			if p.GetOnSuccessWork() != nil {
				p.GetOnSuccessWork()(tasks)
			}
		}
	}

	return nil
}

// SetWorkerCount updates the total number of workers as well as adding/removing workers.
// It returns number of added/removed workers.
func (p *WorkerPool) SetWorkerCount(newCount int) int {
	oldCount := p.workerCount
	diff := newCount - oldCount

	p.mtx.Lock()
	p.workerCount = newCount
	p.mtx.Unlock()

	counter := 0

	if diff < 0 {
		for i := 0; i < diff*-1; i++ {
			p.GetQuitChan() <- true

			if p.GetOnDeleteWorker() != nil {
				p.GetOnDeleteWorker()()
			}

			p.mtx.Lock()
			counter += 1
			p.mtx.Unlock()
		}

	} else if diff > 0 {
		for i := 0; i < diff; i++ {
			go p.Run()

			if p.GetOnNewWorker() != nil {
				p.GetOnNewWorker()()
			}

			p.mtx.Lock()
			counter += 1
			p.mtx.Unlock()
		}
	}

	return counter
}

// GetWorkerCount returns the total number of workers.
func (p *WorkerPool) GetWorkerCount() int {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.workerCount
}

// RetryDuration returns randomized duration for retry sleep.
func (p *WorkerPool) RetryDuration() time.Duration {
	return time.Duration(rand.Intn(p.GetMaxRetryMilliseconds())) * time.Millisecond
}

// SetMaxRetryMilliseconds sets maxRetryMilliseconds
func (p *WorkerPool) SetMaxRetryMilliseconds(ms int) {
	p.mtx.Lock()
	p.maxRetryMilliseconds = ms
	p.mtx.Unlock()
}

// GetMaxRetryMilliseconds returns maxRetryMilliseconds
func (p *WorkerPool) GetMaxRetryMilliseconds() int {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.maxRetryMilliseconds
}

// SetRetries sets the number of retries.
func (p *WorkerPool) SetRetries(retries int) {
	p.mtx.Lock()
	p.retries = retries
	p.mtx.Unlock()
}

// GetRetries returns the number of retries.
func (p *WorkerPool) GetRetries() int {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.retries
}

// SetBatchSize sets the batch size.
func (p *WorkerPool) SetBatchSize(batchSize int) {
	p.mtx.Lock()
	p.batchSize = batchSize
	p.mtx.Unlock()
}

// GetBatchSize returns the batch size.
func (p *WorkerPool) GetBatchSize() int {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.batchSize
}

// SetQuitChan sets the quit channel.
func (p *WorkerPool) SetQuitChan(quitChan chan bool) {
	p.mtx.Lock()
	p.quitChan = quitChan
	p.mtx.Unlock()
}

// SetQuitChan returns the quit channel.
func (p *WorkerPool) GetQuitChan() chan bool {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.quitChan
}

// ResizeInChan destroys the old imput channel and create a new input channel with the new size.
func (p *WorkerPool) ResizeInChan(inChanSize int) {
	// 1. Kill all existing workers
	if p.GetWorkerCount() > 0 {
		for i := 0; i < p.GetWorkerCount(); i++ {
			p.GetQuitChan() <- true
		}
	}

	// 2. Close old channel
	if p.GetInChan() != nil {
		close(p.inChan)
	}

	// 3. Create a new channel
	// if size > 0, create a buffered channel
	// if size == 0, create an unbuffered channel
	var newChan chan interface{}
	if inChanSize > 0 {
		newChan = make(chan interface{}, inChanSize)
	} else {
		newChan = make(chan interface{})
	}

	p.mtx.Lock()
	p.inChan = newChan
	p.inChanSize = inChanSize
	p.mtx.Unlock()

	// 4. Spawn new workers
	for i := 0; i < p.GetWorkerCount(); i++ {
		go p.Run()
	}
}

// GetInChan returns the input channel.
func (p *WorkerPool) GetInChan() chan interface{} {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.inChan
}

// GetInChanSize returns the size of input channel.
func (p *WorkerPool) GetInChanSize() int {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.inChanSize
}

// SetWorker sets worker (defined as function hook).
func (p *WorkerPool) SetWorker(worker func([]interface{}) error) {
	p.mtx.Lock()
	p.worker = worker
	p.mtx.Unlock()
}

// GetWorker returns worker (defined as function hook).
func (p *WorkerPool) GetWorker() func([]interface{}) error {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	return p.worker
}
