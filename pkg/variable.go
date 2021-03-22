package pkg

import (
	"github.com/kashmir/internal"
	"sync/atomic"
)

type StmVariable struct {
	val  atomic.Value
	lock internal.VersionedLock
}

func NewStmVariable(value interface{}) *StmVariable {
	stmVariable := &StmVariable{
		val:  atomic.Value{},
		lock: 0,
	}
	stmVariable.val.Store(value)
	return stmVariable
}
