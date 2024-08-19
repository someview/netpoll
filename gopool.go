package netpoll

import (
	"github.com/panjf2000/ants/v2"
)

var goroutinePool, _ = ants.NewPool(-1)

// Go 在协程池中运行Func
func Go(f func()) {
	_ = goroutinePool.Submit(f)
}
