package internal

import (
	"github.com/pkg/errors"
	"sync/atomic"
)

const versionOffset = 63

var (
	errLockModified    = errors.New("lock has been modified")
	errAlreadyLocked   = errors.New("lock is already locked")
	errAlreadyReleased = errors.New("lock is already released")
)

// VersionedLock consists of a lock bit and a version number.
// Note that this lock doesn't enforce ownership!
type VersionedLock uint64

// Tries to acquire lock.
// Non-blocking.
func (vl *VersionedLock) TryAcquire() error {
	currentlyLocked, currentVersion, currentLock := vl.Sample()
	if currentlyLocked {
		return errAlreadyLocked
	}

	// Lock = true; Version = current
	return vl.tryCompareAndSwap(true, currentVersion, currentLock)
}

// Releases lock.
func (vl *VersionedLock) Release() error {
	currentlyLocked, currentVersion, currentLock := vl.Sample()
	if !currentlyLocked {
		return errAlreadyReleased
	}

	// Lock = false; Version = current
	return vl.tryCompareAndSwap(false, currentVersion, currentLock)
}

// Atomically updates lock version and releases it.
func (vl *VersionedLock) VersionedRelease(newVersion uint64) error {
	currentlyLocked, _, currentLock := vl.Sample()
	if !currentlyLocked {
		return errAlreadyReleased
	}

	// Lock = false; Version = new
	return vl.tryCompareAndSwap(false, newVersion, currentLock)
}

// Retrieves lock state.
func (vl *VersionedLock) Sample() (bool, uint64, uint64) {
	current := atomic.LoadUint64((*uint64)(vl))
	locked, version := vl.parse(current)
	return locked, version, current
}

func (vl *VersionedLock) tryCompareAndSwap(doLock bool, desiredVersion uint64, compareTo uint64) error {
	newLock, err := vl.serialize(doLock, desiredVersion)
	if err != nil {
		return errors.WithMessage(err, "serialize new unlocked lock")
	}

	if swapped := atomic.CompareAndSwapUint64((*uint64)(vl), compareTo, newLock); !swapped {
		return errLockModified
	}
	return nil
}

func (vl *VersionedLock) serialize(locked bool, version uint64) (uint64, error) {
	if (version >> versionOffset) == 1 { // Version mustn't override our lock bit.
		return 0, errors.Errorf("version number cannot be larger than (2^63)-1 (got: %d)", version)
	}

	if locked {
		return (1 << versionOffset) | version, nil
	}
	return version << versionOffset, nil
}

func (vl *VersionedLock) parse(serialized uint64) (bool, uint64) {
	version := (1<<versionOffset - 1) & serialized
	lockedBit := serialized >> versionOffset
	return lockedBit == 1, version
}
