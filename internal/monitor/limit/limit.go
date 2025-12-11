package limit

import (
	"sync"
	"time"
)

type Limiter struct {
	quota      float64
	maxQuota   float64
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

func New(maxQuota, refillRate float64) *Limiter {
	return &Limiter{
		quota:      maxQuota,
		maxQuota:   maxQuota,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.lastRefill).Seconds()
	l.quota += elapsed * l.refillRate

	if l.quota > l.maxQuota {
		l.quota = l.maxQuota
	}

	l.lastRefill = now

	if l.quota >= 1.0 {
		l.quota -= 1.0
		return true
	}

	return false
}
