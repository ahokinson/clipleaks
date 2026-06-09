package util

import "testing"

func TestPointer(t *testing.T) {
	i := Pointer(42)
	if i == nil || *i != 42 {
		t.Fatalf("Pointer(42) = %v, want pointer to 42", i)
	}

	s := Pointer("hello")
	if s == nil || *s != "hello" {
		t.Fatalf("Pointer(\"hello\") = %v, want pointer to \"hello\"", s)
	}

	// Each call returns a distinct address.
	a, b := Pointer(1), Pointer(1)
	if a == b {
		t.Error("Pointer should allocate a new value per call")
	}
}
