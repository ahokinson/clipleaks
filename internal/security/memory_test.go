package security

import (
	"bytes"
	"testing"
)

func TestSecureString_BasicAccessors(t *testing.T) {
	s := New("hunter2")
	if s.String() != "hunter2" {
		t.Errorf("String() = %q, want %q", s.String(), "hunter2")
	}
	if s.Len() != len("hunter2") {
		t.Errorf("Len() = %d, want %d", s.Len(), len("hunter2"))
	}
	if !bytes.Equal(s.Bytes(), []byte("hunter2")) {
		t.Errorf("Bytes() = %v, want %v", s.Bytes(), []byte("hunter2"))
	}
}

func TestSecureString_WipeClearsAndNils(t *testing.T) {
	s := New("topsecret")
	s.Wipe()

	if s.Len() != 0 {
		t.Errorf("Len() after Wipe = %d, want 0", s.Len())
	}
	if s.String() != "" {
		t.Errorf("String() after Wipe = %q, want empty", s.String())
	}
	if s.Bytes() != nil {
		t.Errorf("Bytes() after Wipe = %v, want nil", s.Bytes())
	}
}

func TestSecureString_NilSafety(t *testing.T) {
	var s *SecureString
	if s.String() != "" {
		t.Error("nil SecureString String() should be empty")
	}
	if s.Len() != 0 {
		t.Error("nil SecureString Len() should be 0")
	}
	if s.Bytes() != nil {
		t.Error("nil SecureString Bytes() should be nil")
	}
	s.Wipe() // must not panic
}

func TestWipeBytes(t *testing.T) {
	data := []byte("secret")
	WipeBytes(data)
	for i, b := range data {
		if b != 0 {
			t.Errorf("byte %d not zeroed: %d", i, b)
		}
	}

	WipeBytes(nil) // must not panic
}
