package fab

import "context"

// HashDB is the type of a database for storing hashes.
// It must permit concurrent operations safely.
// It may expire entries to save space.
type HashDB interface {
	// Has tells whether the database contains the given entry.
	Has(context.Context, []byte) (bool, error)

	// Add adds an entry to the database.
	Add(context.Context, []byte) error
}
