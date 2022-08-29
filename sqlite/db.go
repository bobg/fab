package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"time"

	"github.com/benbjohnson/clock"
	_ "github.com/mattn/go-sqlite3" // to get the "sqlite3" driver for sql.Open
	"github.com/pkg/errors"
	"github.com/pressly/goose/v3"

	"github.com/bobg/fab"
)

// DB is an implementation of fab.HashDB that uses a Sqlite3 file for persistent storage.
type DB struct {
	db             *sql.DB
	keep           time.Duration
	clk            clock.Clock
	updateOnAccess bool
}

var _ fab.HashDB = &DB{}

//go:embed migrations/*.sql
var migrations embed.FS

// Open opens the given file and returns it as a *DB.
// The file is created if it doesn't already exist.
// Callers should call Close when finished operating on the database.
func Open(ctx context.Context, path string, opts ...Option) (*DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, errors.Wrapf(err, "opening sqlite db %s", path)
	}

	goose.SetBaseFS(migrations)
	if err = goose.SetDialect("sqlite3"); err != nil {
		return nil, errors.Wrap(err, "setting migration dialect")
	}
	if err = goose.Up(db, "migrations"); err != nil {
		return nil, errors.Wrap(err, "executing db migrations")
	}

	result := &DB{
		db:             db,
		updateOnAccess: true,
	}
	for _, opt := range opts {
		opt(result)
	}
	if result.clk == nil {
		result.clk = clock.New()
	}
	return result, nil
}

// Close releases the resources of s.
func (db *DB) Close() error {
	return db.db.Close()
}

// Option is the type of a config option that can be passed to Open.
type Option func(*DB)

// Keep is an Option that sets the amount of time to keep a database entry.
// By default, DB keeps all entries.
// Using Keep(d) allows DB to evict entries whose last-access time is older than d.
func Keep(d time.Duration) Option {
	return func(db *DB) {
		db.keep = d
	}
}

// WithClock is an Option that sets the database's clock.
// By default it's clock.New(),
// i.e. the normal time-telling clock.
// For testing this can be set to a mock clock.
func WithClock(clk clock.Clock) Option {
	return func(db *DB) {
		db.clk = clk
	}
}

// UpdateOnAccess is an Option controlling whether to update a db entry's timestamp when accessed with Has.
// The default is true: each Has of a value refreshes its timestamp to prevent its expiration.
func UpdateOnAccess(update bool) Option {
	return func(db *DB) {
		db.updateOnAccess = update
	}
}

// Has tells whether db contains the given hash.
// If found, it also updates the last-access time of the hash.
func (db *DB) Has(ctx context.Context, h []byte) (bool, error) {
	if db.updateOnAccess {
		const q = `UPDATE hashes SET unix_secs = $1 WHERE hash = $2`
		now := db.clk.Now()
		res, err := db.db.ExecContext(ctx, q, now.Unix(), h)
		if err != nil {
			return false, errors.Wrap(err, "updating database")
		}
		aff, err := res.RowsAffected()
		return aff > 0, errors.Wrap(err, "counting affected rows")
	}

	const q = `SELECT COUNT(*) FROM hashes WHERE hash = $1`
	var count int
	err := db.db.QueryRowContext(ctx, q, h).Scan(&count)
	if err != nil {
		return false, errors.Wrap(err, "querying database")
	}
	return count > 0, nil
}

// Add adds a hash to db.
// If it is already present, its last-access time is updated.
// If db was opened with the Keep option,
// entries with old last-access times are evicted.
func (db *DB) Add(ctx context.Context, h []byte) error {
	const q = `INSERT INTO hashes (hash, unix_secs) VALUES ($1, $2) ON CONFLICT DO UPDATE SET unix_secs = $2 WHERE hash = $1`
	now := db.clk.Now()
	_, err := db.db.ExecContext(ctx, q, h, now.Unix())
	if err != nil {
		return errors.Wrap(err, "adding hash to database")
	}
	if db.keep > 0 {
		const q2 = `DELETE FROM hashes WHERE unix_secs < $1`
		when := now.Add(-db.keep).Unix()
		_, err = db.db.ExecContext(ctx, q2, when)
		if err != nil {
			return errors.Wrap(err, "evicting expired database entries")
		}
	}
	return nil
}
