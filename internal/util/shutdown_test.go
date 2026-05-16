package util

import (
	"errors"
	"testing"
)

func TestManager_RegisterAndShutdown(t *testing.T) {
	m := NewManager()

	var order []int
	m.Register(func() error {
		order = append(order, 1)
		return nil
	})
	m.Register(func() error {
		order = append(order, 2)
		return nil
	})
	m.Register(func() error {
		order = append(order, 3)
		return nil
	})

	m.Shutdown()

	// Handlers execute in reverse order (LIFO).
	want := []int{3, 2, 1}
	if len(order) != len(want) {
		t.Fatalf("got %d calls, want %d", len(order), len(want))
	}
	for i, v := range want {
		if order[i] != v {
			t.Errorf("order[%d] = %d, want %d", i, order[i], v)
		}
	}
}

func TestManager_ShutdownIdempotent(t *testing.T) {
	m := NewManager()

	var count int
	m.Register(func() error {
		count++
		return nil
	})

	m.Shutdown()
	m.Shutdown()
	m.Shutdown()

	if count != 1 {
		t.Fatalf("handler called %d times, want 1 (idempotent)", count)
	}
}

func TestManager_HandlerErrorLogged(t *testing.T) {
	m := NewManager()

	m.Register(func() error {
		return errors.New("expected error")
	})

	// Should not panic; error is logged.
	m.Shutdown()
}

func TestManager_Empty(t *testing.T) {
	m := NewManager()
	// Should not panic with no handlers.
	m.Shutdown()
}
