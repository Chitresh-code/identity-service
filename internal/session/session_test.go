package session

import "testing"

func TestNewToken(t *testing.T) {
	raw1, hash1, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken() error = %v", err)
	}
	raw2, hash2, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken() error = %v", err)
	}

	if raw1 == raw2 {
		t.Fatalf("NewToken() produced the same raw token twice: %q", raw1)
	}
	if hash1 == hash2 {
		t.Fatalf("NewToken() produced the same hash twice: %q", hash1)
	}
	if hash1 != HashToken(raw1) {
		t.Fatalf("HashToken(%q) = %q, want %q (the hash NewToken returned)", raw1, HashToken(raw1), hash1)
	}
}

func TestHashTokenDeterministic(t *testing.T) {
	if HashToken("same-input") != HashToken("same-input") {
		t.Fatal("HashToken is not deterministic for the same input")
	}
	if HashToken("a") == HashToken("b") {
		t.Fatal("HashToken produced the same hash for different inputs")
	}
}
