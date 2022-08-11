package fab

import "sync"

// A gate is a synchronization structure that can be open or closed.
// Waiting on a closed gate blocks until someone opens it.
// Waiting on an open gate succeeds immediately.
type gate struct {
	c    *sync.Cond
	open bool
}

func newGate(open bool) *gate {
	return &gate{
		c:    sync.NewCond(new(sync.Mutex)),
		open: open,
	}
}

// Sets the gate to open or closed.
func (g *gate) set(open bool) {
	g.c.L.Lock()
	wasOpen := g.open
	g.open = open
	if open && !wasOpen {
		g.c.Broadcast()
	}
	g.c.L.Unlock()
}

func (g *gate) wait() {
	g.c.L.Lock()
	for !g.open {
		g.c.Wait()
	}
	g.c.L.Unlock()
}
