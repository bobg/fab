package fab

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/bobg/errors"

	"github.com/bobg/fab/sqlite"
)

// HashDB is the type of a database for storing hashes.
// It must permit concurrent operations safely.
// It may expire entries to save space.
type HashDB interface {
	// Has tells whether the database contains the given entry.
	Has(context.Context, []byte) (bool, error)

	// Add adds an entry to the database.
	Add(context.Context, []byte) error
}

// OpenHashDB ensures the given directory exists and opens (or creates) the hash DB there.
// Callers must make sure to call Close on the returned DB when finished with it.
func OpenHashDB(dir string) (*sqlite.DB, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, errors.Wrapf(err, "creating directory %s", dir)
	}
	dbfile := filepath.Join(dir, "hash.db")
	db, err := sqlite.Open(dbfile, sqlite.Keep(30*24*time.Hour)) // keep db entries for 30 days
	return db, errors.Wrapf(err, "opening file %s", dbfile)
}
