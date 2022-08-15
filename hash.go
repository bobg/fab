package fab

import (
	"context"

	"github.com/pkg/errors"
)

// HashTarget is a Target that knows how to produce a hash
// (or "digest")
// representing the complete state of the target:
// the inputs, the outputs, and the rules for turning one into the other.
// Any change in any of those should produce a distinct hash value.
//
// The Run method of HashTarget can use this hash
// to determine whether the outputs are up to date
// with respect to the inputs and build rules,
// without resorting to comparing file modification times,
// which aren't completely reliable for this purpose.
// For this to work,
// the context passed to Run must be decorated with a HashDB
// using WithHashDB.
type HashTarget struct {
	Target Target
	Hash   func(context.Context) ([]byte, error)
}

var _ Target = HashTarget{}

// Run implements Target.Run.
// If `ctx` contains a HashDB,
// then it is consulted to see whether ht.Hash() is in it.
// If it is, then ht's outputs are up to date and Run succeeds trivially.
// Otherwise ht.Target's Run method is invoked.
// If that succeeds, ht's new hash is added to the HashDB.
func (ht HashTarget) Run(ctx context.Context) error {
	if ht.Hash == nil {
		return ht.Target.Run(ctx)
	}
	db := GetHashDB(ctx)
	if db == nil {
		return ht.Target.Run(ctx)
	}
	h, err := ht.Hash(ctx)
	if err != nil {
		return errors.Wrapf(err, "computing hash for %s", ht.ID())
	}
	has, err := db.Has(ctx, h)
	if err != nil {
		return errors.Wrapf(err, "checking hash db for hash of %s", ht.ID())
	}
	if has {
		// Up to date.
		return nil
	}
	err = Run(ctx, ht)
	if err != nil {
		return errors.Wrapf(err, "running %s", ht.ID())
	}
	h, err = ht.Hash(ctx)
	if err != nil {
		return errors.Wrapf(err, "computing updated hash for %s", ht.ID())
	}
	err = db.Add(ctx, h)
	return errors.Wrap(err, "updating hash db")
}

// ID implements Target.ID.
func (ht HashTarget) ID() string {
	return ht.Target.ID()
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
