package ratedr

////////////////////////////////////////////////////////////////////////////////

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

////////////////////////////////////////////////////////////////////////////////
func TestBasic(t *testing.T) {
	const expiryDuration = 20 * time.Second
	const gcInterval = 10 * time.Second

	r, err := NewRateDr(expiryDuration)
	assert.Nil(t, err)

	assert.Nil(t, r.RegisterContext("foo", 10, 3))
	assert.Nil(t, r.RegisterContext("bar", 20, 10))

	done := make(chan struct{})
	go r.GC(gcInterval, done)

	assert.True(t, r.IsAllowed("foo", "IP_0"))
	assert.True(t, r.IsAllowed("foo", "IP_0"))
	assert.True(t, r.IsAllowed("foo", "IP_0"))
	assert.False(t, r.IsAllowed("foo", "IP_0"))
	<-time.After(1 * time.Second)
	assert.True(t, r.IsAllowed("foo", "IP_0"))
	assert.True(t, r.IsAllowed("foo", "IP_0"))
	assert.True(t, r.IsAllowed("foo", "IP_0"))
	assert.False(t, r.IsAllowed("foo", "IP_0"))
	<-time.After(100 * time.Millisecond)
	assert.True(t, r.IsAllowed("foo", "IP_0"))
	assert.False(t, r.IsAllowed("foo", "IP_0"))

	done <- struct{}{}
}
