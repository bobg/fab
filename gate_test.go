package fab

import "testing"

func TestGate(t *testing.T) {
	t.Parallel()

	opened := false

	g := newGate(false)

	done := make(chan struct{})

	go func() {
		g.wait()
		opened = true
		close(done)
	}()

	if opened {
		t.Error("opened is true too soon")
	}
	g.set(true)
	<-done
	if !opened {
		t.Error("opened is not false yet")
	}
}
