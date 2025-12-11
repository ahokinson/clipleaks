package security

import (
	"crypto/rand"
)

type SecureString struct {
	data []byte
}

func New(s string) *SecureString {
	data := make([]byte, len(s))
	copy(data, s)
	return &SecureString{data: data}
}

func (s *SecureString) String() string {
	if s == nil || s.data == nil {
		return ""
	}
	return string(s.data)
}

func (s *SecureString) Bytes() []byte {
	if s == nil {
		return nil
	}
	return s.data
}

func (s *SecureString) Len() int {
	if s == nil || s.data == nil {
		return 0
	}
	return len(s.data)
}

func (s *SecureString) Wipe() {
	if s == nil || s.data == nil {
		return
	}

	_, _ = rand.Read(s.data)

	for i := range s.data {
		s.data[i] = 0
	}

	s.data = nil
}

func WipeBytes(data []byte) {
	if data == nil {
		return
	}

	_, _ = rand.Read(data)

	for i := range data {
		data[i] = 0
	}
}
