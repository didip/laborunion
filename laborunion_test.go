package laborunion

import (
	"sync/atomic"
	"testing"
)

func Test_BasicInit(t *testing.T) {
	p := New()
	p.SetBatchSize(20)
	p.ResizeInChan(1000)
	p.SetWorker(func([]interface{}, interface{}) error {
		return nil
	})

	if p.GetInChan() == nil {
		t.Fatalf("After resizing InChan, it should not be nil")
	}
	if cap(p.GetInChan()) != 1000 {
		t.Fatalf("Failed to resize InChan correctly")
	}

	p.SetWorkerCount(3)
	if p.GetWorkerCount() != 3 {
		t.Fatalf("Failed to spawn 3 workers")
	}
}

func Test_BasicHooks(t *testing.T) {
	p := New()
	p.SetBatchSize(1)
	p.ResizeInChan(1000)
	p.SetWorker(func([]interface{}, interface{}) error {
		return nil
	})

	onNewWorker := false
	p.SetOnNewWorker(func() {
		onNewWorker = true
	})

	onDeleteWorker := false
	p.SetOnDeleteWorker(func() {
		onDeleteWorker = true
	})

	beforeBatchingHookCalled := false
	p.SetBeforeBatchingHook(func() {
		beforeBatchingHookCalled = true
	})

	p.SetWorkerCount(1)
	if !onNewWorker {
		t.Fatalf("Failed to trigger onNewWorker hook")
	}

	p.SetWorkerCount(0)
	if !onDeleteWorker {
		t.Fatalf("Failed to trigger onDeleteWorker hook")
	}

	p.SetWorkerCount(1)

	p.GetInChan() <- true

	if !beforeBatchingHookCalled {
		t.Fatalf("Failed to trigger beforeBatchingHookCalled hook")
	}
}

func Test_BasicSpawnDespawn(t *testing.T) {
	p := New()
	p.SetWorker(func([]interface{}, interface{}) error {
		return nil
	})

	changes := p.SetWorkerCount(10)
	if changes != 10 {
		t.Errorf("Expected to spawn 10 workers. Got: %v", changes)
	}

	if p.GetWorkerCount() != 10 {
		t.Errorf("Expected to have 10 workers. Got: %v", p.GetWorkerCount())
	}

	changes = p.SetWorkerCount(7)
	if changes != 3 {
		t.Errorf("Expected to reduce 3 workers. Got: %v", changes)
	}

	if p.GetWorkerCount() != 7 {
		t.Errorf("Expected to have 7 workers. Got: %v", p.GetWorkerCount())
	}
}

func Test_BasicSetterGetter(t *testing.T) {
	p := New()
	p.SetWorker(func([]interface{}, interface{}) error {
		return nil
	})

	p.SetWorkerCount(2)
	if p.GetWorkerCount() != 2 {
		t.Errorf("Failed to set number of workers. Got: %v", p.GetWorkerCount())
	}

	if p.RetryDuration() <= 0 {
		t.Errorf("Failed to set default retry duration")
	}

	p.SetRetries(2)
	if p.GetRetries() != 2 {
		t.Errorf("Failed to set number of workers. Got: %v", p.GetRetries())
	}
}

func Test_BasicConsumption(t *testing.T) {
	var workDone int64
	expectedWorkDone := int64(10)

	p := New()
	p.SetWorker(func(tasks []interface{}, task interface{}) error {
		atomic.AddInt64(&workDone, 1)
		println(task.(int64))
		return nil
	})

	changes := p.SetWorkerCount(1)
	if changes != 1 {
		t.Fatalf("Expected to spawn 1 workers. Got: %v", changes)
	}

	for i := int64(0); i < expectedWorkDone; i++ {
		p.GetInChan() <- i
	}

	finalWorkDone := atomic.LoadInt64(&workDone)

	// This test failed in a weird way, we clearly performed the work 10 times, but recorded only 9 times.
	if finalWorkDone != expectedWorkDone {
		t.Fatalf("Failed to process all %v jobs. Completed only: %v", expectedWorkDone, finalWorkDone)
	}
}
