package fab

import "context"

// HashTarget is a Target that knows how to produce a hash
// (or "digest")
// representing the complete state of the target:
// the inputs, the outputs, and the rules for turning one into the other.
// Any change in any of those should produce a distinct hash value.
//
// When a HashTarget is executed by Runner.Run,
// its Run method is skipped
// and it succeeds trivially
// if the Runner can determine
// that the outputs are up to date
// with respect to the inputs and build rules.
// It does this by consulting a HashDB
// that is populated with the hashes of HashTargets
// whose Run methods succeeded in the past.
//
// Using such a hash to decide whether a target's outputs are up to date
// is preferable to using file modification times
// (like Make does, for example).
// Those aren't always sufficient for this purpose,
// nor are they entirely reliable,
// considering the limited resolution of filesystem timestamps,
// the possibility of clock skew, etc.
type HashTarget interface {
	Target
	Hash(context.Context) ([]byte, error)
}

// HashDB is the type of a database for storing hashes.
// It must permit concurrent operations safely.
// It may expire entries to save space.
type HashDB interface {
	// Has tells whether the database contains the given entry.
	Has(context.Context, []byte) (bool, error)

	// Add adds an entry to the database.
	Add(context.Context, []byte) error
}
