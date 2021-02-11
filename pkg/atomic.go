package pkg

import "github.com/kashmir/internal"

// todo: lock readset?
// todo: version clock overflow?

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

		lockSet := make(map[*StmVariable]int, 0)
		if acquiredWriteSet := tryAcquireWriteSet(ctx, lockSet); !acquiredWriteSet {
			continue
		}

		ctx.writeVersion = versionClock.Increment()

		// todo: lock readset?
	}
}

func tryAcquireWriteSet(ctx *StmContext, writeLockSet map[*StmVariable]int) bool {
	for writeVal := range ctx.writeLog {
		if err := writeVal.lock.TryAcquire(); err != nil {
			// todo: log error?
			releaseLockSet(writeLockSet)
			return false
		}

		writeLockSet[writeVal] = 1
	}
	return true
}

func releaseLockSet(writeLockSet map[*StmVariable]int) {
	for alreadyLocked := range writeLockSet {
		if err := alreadyLocked.lock.Release(); err != nil {
			panic(err)
		}
	}
}
