package apikey

import "testing"

func TestNewKey(t *testing.T) {
	plaintext1, prefix1, hash1, err := NewKey()
	if err != nil {
		t.Fatalf("NewKey() error = %v", err)
	}
	plaintext2, prefix2, hash2, err := NewKey()
	if err != nil {
		t.Fatalf("NewKey() error = %v", err)
	}

	if plaintext1 == plaintext2 {
		t.Fatalf("NewKey() produced the same plaintext twice: %q", plaintext1)
	}
	if prefix1 == prefix2 {
		t.Fatalf("NewKey() produced the same prefix twice: %q", prefix1)
	}
	if hash1 == hash2 {
		t.Fatalf("NewKey() produced the same hash twice: %q", hash1)
	}
	if len(plaintext1) <= len(prefix1) || plaintext1[:len(prefix1)] != prefix1 {
		t.Fatalf("NewKey() plaintext %q does not start with its own prefix %q", plaintext1, prefix1)
	}
}
