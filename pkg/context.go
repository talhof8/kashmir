package pkg

type StmContext struct {
	readLog      map[*StmVariable]interface{}
	writeLog     map[*StmVariable]interface{}
	restart      bool
	readVersion  uint64
	writeVersion uint64
}

func (sc *StmContext) Write(stmVariable *StmVariable, newVal interface{}) {
	sc.writeLog[stmVariable] = newVal
}

func (sc *StmContext) Read(stmVariable *StmVariable) interface{} {
	if newVal, foundInWriteLog := sc.writeLog[stmVariable]; foundInWriteLog { // Short road to success...
		return newVal
	}

	_, preReadVersion, _ := stmVariable.lock.Sample()
	readVal := stmVariable.val.Load()
	locked, postReadVersion, _ := stmVariable.lock.Sample()

	// Fail transaction if:
	// 1. Variable is currently being changed by some other goroutine; or if
	// 2. Variable was changed before/after being read; or if
	// 3. Variable is too new meaning our read version is outdated
	sc.restart = locked || preReadVersion != postReadVersion || preReadVersion > sc.readVersion

	return readVal
}
