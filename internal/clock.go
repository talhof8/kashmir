package internal

import "sync/atomic"

// VersionClock represents a global inter-transactional clock.
type VersionClock uint64

// Atomically increments clock and retrieves new value.
func (vc *VersionClock) Increment() uint64 {
	return atomic.AddUint64((*uint64)(vc), 1)
}

// Atomically retrieves current clock value.
func (vc *VersionClock) Load() uint64 {
	return atomic.LoadUint64((*uint64)(vc))
}
