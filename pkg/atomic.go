package pkg

import (
	"github.com/kashmir/internal"
	"github.com/pkg/errors"
)

var versionClock internal.VersionClock

func StmAtomic(block func(*StmContext) interface{}) interface{} {
	for {
		ctx := &StmContext{
			readLog:      make(map[*StmVariable]interface{}, 0),
			writeLog:     make(map[*StmVariable]interface{}, 0),
			restart:      false,
			readVersion:  versionClock.Load(),
			writeVersion: 0,
		}

		retVal := block(ctx)
		if ctx.restart {
			continue
		}

		// "And she's buying a stairway to heaven..."
		if len(ctx.writeLog) == 0 {
			return retVal
		}

		lockSet := make(map[*StmVariable]int, 0)
		if err := tryAcquireSets(ctx, lockSet); err != nil {
			if fatal := isFatalAcquireErr(err); fatal { // Avoid a panic if lock is already acquired.
				panic(fatal)
			}
			continue
		}

		ctx.writeVersion = versionClock.Increment()

		// Now that our read and write sets are locked, we need to ensure that nothing has changed in terms of our
		// read set, in-between running the user's code and locking everything.
		// However, if no other concurrent actors were involved (readVersion == writeVersion - 1), there is no need to
		// validate anything cause we were all alone.
		if ctx.readVersion != ctx.writeVersion-1 {
			if validated := validateReadSet(ctx, lockSet); !validated {
				continue
			}
		}

		commitTransaction(ctx, lockSet)

		return retVal
	}
}

func tryAcquireSets(ctx *StmContext, lockSet map[*StmVariable]int) error {
	for writeVar := range ctx.writeLog {
		if err := writeVar.lock.TryAcquire(); err != nil {
			releaseLockSet(lockSet)
			return errors.WithMessage(err, "try acquire write log")
		}

		lockSet[writeVar] = 1
	}

	for readVar := range ctx.readLog {
		// Avoid locking a variable which was already locked by us (either by previously being read
		// or by being part of the write log).
		if _, alreadyLocked := lockSet[readVar]; alreadyLocked {
			continue
		}

		if err := readVar.lock.TryAcquire(); err != nil {
			releaseLockSet(lockSet)
			return errors.WithMessage(err, "try acquire read log")
		}

		lockSet[readVar] = 1
	}
	return nil
}

func releaseLockSet(writeLockSet map[*StmVariable]int) {
	for alreadyLocked := range writeLockSet {
		if err := alreadyLocked.lock.Release(); err != nil {
			panic(err)
		}
	}
}

func validateReadSet(ctx *StmContext, lockSet map[*StmVariable]int) bool {
	for readVar := range ctx.readLog {
		locked, version, _ := readVar.lock.Sample()
		_, lockedByUs := lockSet[readVar]
		if (locked && !lockedByUs) || version > ctx.readVersion {
			return false
		}
	}

	return true
}

func commitTransaction(ctx *StmContext, lockSet map[*StmVariable]int) {
	releasedSet := make(map[*StmVariable]int, len(lockSet))

	for writeVar, writeVal := range ctx.writeLog {
		oldVal := writeVar.val.Load()
		writeVar.val.Store(writeVal)

		if err := writeVar.lock.VersionedRelease(ctx.writeVersion); err != nil {
			writeVar.val.Store(oldVal) // Just in case a recover is used up the latter.
			panic(err)
		}

		releasedSet[writeVar] = 1
	}

	for readVar := range ctx.readLog {
		// Avoid releasing a variable which was already released by us (either by previously being read
		// or by being part of the write log).
		if _, alreadyReleased := releasedSet[readVar]; alreadyReleased {
			continue
		}

		if err := readVar.lock.Release(); err != nil {
			panic(err)
		}

		releasedSet[readVar] = 1
	}
}

func isFatalAcquireErr(err error) bool {
	switch errors.Cause(err) {
	case internal.ErrLockModified, internal.ErrVersionOverflow:
		return true
	default:
		return false
	}
}
