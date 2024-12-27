package fab

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/bobg/errors"

	"github.com/bobg/fab/sqlite"
)

// Main is the structure whose Run methods implements the main logic of the fab command.
type Main struct {
	// Fabdir is where to find the user's hash DB, e.g. $HOME/.cache/fab.
	Fabdir string

	// Topdir is the directory containing a _fab subdir or top-level fab.yaml file.
	// If this is not specified, it will be computed by traversing upward from the current directory.
	Topdir string

	// Verbose tells whether to run the driver in verbose mode
	// (by supplying the -v command-line flag).
	Verbose bool

	// List tells whether to run the driver in list-targets mode
	// (by supplying the -list command-line flag).
	List bool

	// DryRun tells whether to run targets in "dry run" mode - i.e., with state-changing operations (like file creation and updating) suppressed.
	DryRun bool

	// Args contains the additional command-line arguments to pass to the driver, e.g. target names.
	Args []string
}

// Run executes the main logic of the fab command.
// It is invoked with the command-line arguments indicated by the fields of m.
// Typically this will include one or more target names,
// in which case the driver will execute the associated rules
// as defined by any fab.yaml files.
func (m *Main) Run(ctx context.Context) error {
	if m.Topdir == "" {
		var err error

		m.Topdir, err = TopDir(".")
		if err != nil {
			return errors.Wrap(err, "finding project's top directory")
		}
	}

	con := NewController(m.Topdir)

	if err := con.ReadYAMLFile(""); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return errors.Wrap(err, "reading YAML file")
	}

	if m.List {
		con.ListTargets(os.Stdout)
		return nil
	}

	ctx = WithVerbose(ctx, m.Verbose)
	ctx = WithDryRun(ctx, m.DryRun)

	db, err := OpenHashDB(m.Fabdir)
	if err != nil {
		return errors.Wrap(err, "opening hash db")
	}
	defer db.Close()
	ctx = WithHashDB(ctx, db)

	targets, err := con.ParseArgs(m.Args)
	if err != nil {
		return errors.Wrap(err, "parsing args")
	}

	return con.Run(ctx, targets...)
}

var bolRegex = regexp.MustCompile("^")

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
