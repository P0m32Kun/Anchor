package worker

import (
	"testing"
)

func TestWorkerServer_atCapacity(t *testing.T) {
	ws := &WorkerServer{maxConcurrency: 2}
	if ws.atCapacity() {
		t.Fatal("empty worker should not be at capacity")
	}
	ws.runningTasks.Store(2)
	if !ws.atCapacity() {
		t.Fatal("expected at capacity when running == max")
	}
	ws.maxConcurrency = 0
	if ws.atCapacity() {
		t.Fatal("max_concurrency 0 means unlimited")
	}
}

func TestWorkerServer_runningTasksAccounting(t *testing.T) {
	ws := &WorkerServer{maxConcurrency: 10}
	ws.runningTasks.Add(1)
	ws.runningTasks.Add(1)
	if ws.runningTasks.Load() != 2 {
		t.Fatalf("running = %d, want 2", ws.runningTasks.Load())
	}
	ws.runningTasks.Add(-1)
	if ws.atCapacity() {
		t.Fatal("should not be at capacity with 1/10")
	}
}
