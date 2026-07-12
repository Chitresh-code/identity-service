package store

import "testing"

func TestNewID(t *testing.T) {
	id1, err := newID()
	if err != nil {
		t.Fatalf("newID() error = %v", err)
	}
	id2, err := newID()
	if err != nil {
		t.Fatalf("newID() error = %v", err)
	}

	if id1 == id2 {
		t.Fatalf("newID() produced the same id twice: %q", id1)
	}
	if len(id1) != 36 {
		t.Fatalf("newID() = %q, want a 36-character UUID string", id1)
	}
	if id1[14] != '4' {
		t.Fatalf("newID() = %q, want version nibble 4 at index 14", id1)
	}
}
