# A minute of Fab: Gates

The method [Controller.Run](https://pkg.go.dev/github.com/bobg/fab#Controller.Run)
invokes the `Run` method for one or more targets,
but only for targets that haven’t run already.

The _outcome_ of a target’s run is the error it produced,
if any.
Once a target runs,
its outcome is cached in this `map`:
[link](https://github.com/bobg/fab/blob/7902b498a752c3ca9cb25765464336e0ed402891/controller.go#L26).
The second and subsequent times that a caller asks a Controller to run a target,
its cached outcome is read from this map.

That’s all pretty straightforward.
But Fab runs targets in multiple goroutines,
and `Controller.Run` has to be concurrent-safe.
What should happen when one goroutine requests a run of target X
when target X is still running at the request of some other goroutine?

The second caller should wait for the first one’s outcome.
So should a third caller,
and a fourth,
and a fifth,
etc.

To make this possible,
the outcome record in that `map` includes not only the error value produced by the target,
but a _synchronization primitive_ called a _gate_.

A gate can be “open” or “closed,”
and it can be waited on.
When a goroutine waits on a closed gate,
it stops until some other goroutine opens the gate.
When a goroutine waits on an open gate,
it doesn’t wait at all.

The first goroutine to ask for a run of target X creates a new `outcome` record.
The `gate` inside it starts out closed.
The outcome record is added to the `map`.
[Link](https://github.com/bobg/fab/blob/7902b498a752c3ca9cb25765464336e0ed402891/runner.go#L80-L81).

The second and subsequent goroutines to ask for a run of target X find the `map` entry
and wait for the gate to open.
[Link](https://github.com/bobg/fab/blob/7902b498a752c3ca9cb25765464336e0ed402891/runner.go#L88).

Meanwhile, the first goroutine runs the target
([link](https://github.com/bobg/fab/blob/7902b498a752c3ca9cb25765464336e0ed402891/runner.go#L96)),
updates the outcome record with the resulting error,
and opens the gate.
[Link](https://github.com/bobg/fab/blob/7902b498a752c3ca9cb25765464336e0ed402891/runner.go#L102).

Opening the gate unblocks other goroutines,
which can now read the error result from the outcome record.
[Link](https://github.com/bobg/fab/blob/7902b498a752c3ca9cb25765464336e0ed402891/runner.go#L89).

## Implementation of gate

You may already know about _mutual-exclusion locks,_
usually abbreviated as mutex.
Go defines [sync.Mutex](https://pkg.go.dev/sync#Mutex) in its standard library.

A mutex can be “locked” and “unlocked,”
or “held”
(or “acquired”)
and “released.”
When one goroutine locks it,
no other goroutine can lock it
until the one holding the lock releases it.

A mutex can ensure that two goroutines can’t write to the same data structure at the same time,
which could leave it in a corrupted state,
and that one goroutine can’t read a data structure
while another goroutine is updating it,
because an incomplete update could produce garbage data.

To build a `gate` we use a mutex and a _condition variable._
Go defines [sync.Cond](https://pkg.go.dev/sync#Cond) in its standard library.

A condition variable can be _signaled_ and waited on.
Waiting on a condition variable blocks a goroutine
until some other goroutine signals the condition variable.

So opening a gate is just a matter of signaling the condition variable
(using [Broadcast](https://pkg.go.dev/sync#Cond.Broadcast),
which signals all waiting goroutines,
rather than [Signal](https://pkg.go.dev/sync#Cond.Signal),
which signals just one).
[Link](https://github.com/bobg/fab/blob/7902b498a752c3ca9cb25765464336e0ed402891/gate.go#L26).

And waiting on a gate is just a matter of waiting for that condition variable to get signaled.
[Link](https://github.com/bobg/fab/blob/7902b498a752c3ca9cb25765464336e0ed402891/gate.go#L33-L35).

It is necessary to hold a mutex when waiting on a condition variable;
that’s what `g.c.L.Lock()` is about.
During the wait,
the mutex is released;
and when the wait succeeds,
the mutex is reacquired.

The code waits on the condition variable in a loop
because it’s not guaranteed that the condition it’s waiting for
(i.e., `g.open`)
will be true when `Wait` ends.
So it waits repeatedly until that condition _is_ true.
