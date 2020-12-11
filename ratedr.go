package ratedr

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

////////////////////////////////////////////////////////////////////////////////

type entry struct {
	*rate.Limiter
	expiryDuration time.Duration
	expiryTime     time.Time
}

// Allow returns the underlying `rate.Limiter`s Allow() function after touching
// the expiry time.
func (e *entry) Allow() bool {
	e.expiryTime = time.Now().Add(e.expiryDuration)
	return e.Allow()
}

func newEntry(r, b int, expiresIn time.Duration) *entry {
	return &entry{
		Limiter:        rate.NewLimiter(rate.Limit(r), b),
		expiryDuration: expiresIn,
		expiryTime:     time.Now().Add(expiresIn),
	}
}

////////////////////////////////////////////////////////////////////////////////

type context struct {
	key            string
	expiresIn      time.Duration
	ratePerSecond  int
	burstPerSecond int
}

func newContext(k string, e time.Duration, r, b int) *context {
	return &context{k, e, r, b}
}

////////////////////////////////////////////////////////////////////////////////

// RateDr is a struct which allows multiple contexts to rate limit with
// different rate limit parameters.
type RateDr struct {
	*sync.RWMutex
	contexts map[string]*context
	active   map[string]*entry
}

// NewRateDr returns a new instance of a `&RateDr`.
func NewRateDr() (*RateDr, error) {
	return &RateDr{
		RWMutex:  &sync.RWMutex{},
		contexts: map[string]*context{},
		active:   map[string]*entry{},
	}, nil
}

////////////////////////////////////////////////////////////////////////////////

// RegisterContext sets a context up with it's specific rate and burst policies and eviction policies.
func (r *RateDr) RegisterContext(prefix string, expiresIn time.Duration, ratePerSec, burstPerSec int) error {
	r.Lock()
	defer r.Unlock()

	r.contexts[prefix] = newContext(prefix, expiresIn, ratePerSec, burstPerSec)
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func (r *RateDr) gc() {
	r.Unlock()
	defer r.Lock()

	now := time.Now()
	for k, v := range r.active {
		if now.After(v.expiryTime) {
			delete(r.active, k)
		}
	}
}

// GC is invoked to clear any expired entries.
func (r *RateDr) GC(interval time.Duration, done <-chan struct{}) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			r.gc()
		}
	}
}

////////////////////////////////////////////////////////////////////////////////

// IsAllowed checks to see if we have hit a rate limit for this particular
// context, and key.
func (r *RateDr) IsAllowed(prefix, key string) bool {
	r.Lock()
	defer r.Unlock()

	ctxt, ok := r.contexts[prefix]
	if !ok {
		return false
	}

	k := prefix + "::" + ctxt.key
	if _, ok := r.active[k]; !ok {
		r.active[k] = newEntry(ctxt.ratePerSecond, ctxt.burstPerSecond, ctxt.expiresIn)
	}

	return r.active[k].Allow()
}
