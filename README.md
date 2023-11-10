## Kashmir 
Transactional Locking II (TL2)-inspired STM library for Go, with a slight touch.
On commit time, this library locks the read-set in addition to the write-set in order to prevent potential
data races in the current TL2 algorithm.

See: <https://www.talhoffman.com/2021/03/22/software-transactional-memory/>

### Example
An example for solving the ATM problem of lack of composability:
```golang
func main() {
	accountA := pkg.NewStmVariable(100)
	accountB := pkg.NewStmVariable(0)

	// Transfer 20 from Alice's account to Bob's one.
	transfer := func(ctx *pkg.StmContext) interface{} {
		currA := ctx.Read(accountA).(int)
		currB := ctx.Read(accountB).(int)

		ctx.Write(accountA, currA-20)
		ctx.Write(accountB, currB+20)

		return nil
	}
	pkg.StmAtomic(transfer)

	// Check the balance of accounts of Alice and Bob.
	inquiries := func(ctx *pkg.StmContext) interface{} {
		balance := make(map[*pkg.StmVariable]int)
		balance[accountA] = ctx.Read(accountA).(int)
		balance[accountB] = ctx.Read(accountB).(int)
		return balance
	}
	balance := pkg.StmAtomic(inquiries).(map[*pkg.StmVariable]int)
	fmt.Printf("The account of Alice holds %v.\nThe account of Bob holds %v.",
		balance[accountA], balance[accountB])
}
```

### Caveats
Please be careful when dealing with pointers (including channels, maps, and slices!) and 
use STM variables instead.

In addition, Golang's type system forces us to use `interface{}` and type assertions.

### Contributions
Contributions are always welcome! :heart:

Feel free to open a Pull Request featuring improvements and fixes you see fit.

### License
Unless otherwise noted, the Kashmir source files are distributed under the Apache Version 2.0 license found in the LICENSE file.
