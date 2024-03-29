package fab

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v2/slices"
	"golang.org/x/tools/go/packages"

	"github.com/bobg/fab/sqlite"
)

// Main is the structure whose Run methods implements the main logic of the fab command.
type Main struct {
	// Fabdir is where to find the user's hash DB and compiled binaries, e.g. $HOME/.cache/fab.
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

	// Force tells whether to force recompilation of the driver before running it.
	Force bool

	// DryRun tells whether to run targets in "dry run" mode - i.e., with state-changing operations (like file creation and updating) suppressed.
	DryRun bool

	// Args contains the additional command-line arguments to pass to the driver, e.g. target names.
	Args []string
}

// Run executes the main logic of the fab command.
// A driver binary with a name matching the Go package path of the _fab subdir
// is sought in m.Fabdir.
// If it does not exist,
// or if its corresponding dirhash is wrong
// (i.e., out of date with respect to the user's code),
// or if m.Force is true,
// it is created with Compile.
// It is then invoked with the command-line arguments indicated by the fields of m.
// Typically this will include one or more target names,
// in which case the driver will execute the associated rules
// as defined by the code in _fab
// and by any fab.yaml files.
//
// If there is no _fab directory,
// Run operates in "driverless" mode,
// in which target definitions are found in fab.yaml files only.
func (m *Main) Run(ctx context.Context) error {
	if m.Topdir == "" {
		var err error

		m.Topdir, err = TopDir(".")
		if err != nil {
			return errors.Wrap(err, "finding project's top directory")
		}
	}

	driver, err := m.getDriver(ctx, false)
	if errors.Is(err, errNoDriver) {
		return m.driverless(ctx)
	}
	if err != nil {
		return errors.Wrap(err, "ensuring driver is up to date")
	}

	args := []string{"-fab", m.Fabdir, "-top", m.Topdir}
	if m.Verbose {
		args = append(args, "-v")
	}
	if m.List {
		args = append(args, "-list")
	}
	if m.Force {
		args = append(args, "-f")
	}
	if m.DryRun {
		args = append(args, "-n")
	}
	args = append(args, m.Args...)

	cmd := exec.CommandContext(ctx, driver, args...)
	cmd.Dir = m.Topdir
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	err = cmd.Run()
	return errors.Wrapf(err, "running %s %s", driver, strings.Join(args, " "))
}

var errNoDriver = errors.New("no driver")

func (m *Main) driverless(ctx context.Context) error {
	if m.Verbose {
		fmt.Println("Running in driverless mode")
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
	ctx = WithForce(ctx, m.Force)
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

const fabVersionBasename = "fab-version.json"

// TODO: Remove skipVersionCheck, which is here only to help an old test keep running.
// Update the test instead.
func (m *Main) getDriver(ctx context.Context, skipVersionCheck bool) (_ string, err error) {
	pkgdir := filepath.Join(m.Topdir, "_fab")
	_, err = os.Stat(pkgdir)
	if errors.Is(err, fs.ErrNotExist) {
		return "", errNoDriver
	}
	config := &packages.Config{
		Mode:    LoadMode,
		Context: ctx,
		Dir:     pkgdir,
	}
	pkgs, err := packages.Load(config, ".")
	if errors.Is(err, fs.ErrNotExist) {
		return "", errNoDriver
	}
	if err != nil {
		return "", errors.Wrapf(err, "loading %s", pkgdir)
	}
	if len(pkgs) == 0 {
		return "", errNoDriver
	}
	if len(pkgs) != 1 {
		return "", fmt.Errorf(
			"loaded %d packages in %s, want 1 %v",
			len(pkgs),
			pkgdir,
			slices.Map(pkgs, func(p *packages.Package) string { return p.PkgPath }),
		)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		err = nil
		for _, e := range pkg.Errors {
			err = errors.Join(err, e)
		}
		return "", errors.Wrapf(err, "loading package %s", pkg.Name)
	}

	driverdir := filepath.Join(m.Fabdir, pkg.PkgPath)
	if err = os.MkdirAll(driverdir, 0755); err != nil {
		return "", errors.Wrapf(err, "ensuring directory %s/%s exists", m.Fabdir, pkg.PkgPath)
	}

	var (
		hashfile    = filepath.Join(driverdir, "hash")
		driver      = filepath.Join(driverdir, "fab.bin")
		versionfile = filepath.Join(driverdir, fabVersionBasename)
		compile     bool
		oldhash     []byte
	)

	if m.Force {
		compile = true
		if m.Verbose {
			fmt.Println("Forcing recompilation of driver")
		}
	} else {
		_, err = os.Stat(driver)
		if errors.Is(err, fs.ErrNotExist) {
			compile = true
			if m.Verbose {
				fmt.Println("Compiling driver")
			}
		} else if err != nil {
			return "", errors.Wrapf(err, "statting %s", driver)
		}
	}

	var buildInfo *debug.BuildInfo
	if !compile && !skipVersionCheck {
		compile, buildInfo, err = m.checkVersion(versionfile)
		if err != nil {
			return "", errors.Wrap(err, "checking Fab version")
		}
	}

	if !compile {
		oldhash, err = os.ReadFile(hashfile)
		if errors.Is(err, fs.ErrNotExist) {
			compile = true
			if m.Verbose {
				fmt.Println("Compiling driver")
			}
		} else if err != nil {
			return "", errors.Wrapf(err, "reading %s", hashfile)
		}
	}

	dh := newDirHasher()
	for _, filename := range pkg.GoFiles { // TODO: is this the right set of files?
		if err = addFileToHash(dh, filename); err != nil {
			return "", errors.Wrapf(err, "hashing file %s", filename)
		}
	}
	newhash, err := dh.hash()
	if err != nil {
		return "", errors.Wrapf(err, "computing hash of directory %s", pkgdir)
	}

	if !compile {
		if newhash == string(oldhash) {
			if m.Verbose {
				fmt.Println("Using existing driver")
			}
		} else {
			compile = true
			if m.Verbose {
				fmt.Println("Recompiling driver")
			}
		}
	}

	if !compile {
		return driver, nil
	}

	if err = CompilePackage(ctx, pkg, driver); err != nil {
		return "", errors.Wrapf(err, "compiling driver %s", driver)
	}
	if err = os.WriteFile(hashfile, []byte(newhash), 0644); err != nil {
		return "", errors.Wrapf(err, "writing %s", hashfile)
	}
	if buildInfo != nil {
		v, err := os.OpenFile(versionfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return "", errors.Wrapf(err, "opening %s for writing", versionfile)
		}
		defer func() {
			closeErr := v.Close()
			if err == nil {
				err = errors.Wrapf(closeErr, "closing %s", versionfile)
			}
		}()
		enc := json.NewEncoder(v)
		enc.SetIndent("", "  ")
		if err = enc.Encode(buildInfo); err != nil {
			return "", errors.Wrap(err, "encoding build info")
		}
	}

	return driver, nil
}

func (m *Main) checkVersion(versionfile string) (bool, *debug.BuildInfo, error) {
	newInfo, ok := debug.ReadBuildInfo()
	if !ok {
		if m.Verbose {
			fmt.Println("Could not read build info of running Fab process, will recompile driver")
		}
		_ = os.Remove(versionfile)
		return true, nil, nil
	}

	f, err := os.Open(versionfile)
	if errors.Is(err, fs.ErrNotExist) {
		if m.Verbose {
			fmt.Printf("No %s file, compiling driver\n", fabVersionBasename)
		}
		return true, newInfo, nil
	}
	if err != nil {
		return false, nil, errors.Wrapf(err, "opening %s", versionfile)
	}
	defer f.Close()

	var (
		dec     = json.NewDecoder(f)
		oldInfo debug.BuildInfo
	)

	if err = dec.Decode(&oldInfo); err != nil {
		if m.Verbose {
			fmt.Printf("Error decoding %s, will recompile driver: %s\n", versionfile, err)
		}
		return true, newInfo, nil
	}

	if !reflect.DeepEqual(*newInfo, oldInfo) {
		if m.Verbose {
			fmt.Println("Fab build info changed, will recompile driver")
		}
		return true, newInfo, nil
	}

	return false, nil, nil
}

func addFileToHash(dh *dirHasher, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return errors.Wrapf(err, "opening %s", filename)
	}
	defer f.Close()

	return dh.file(filename, f)
}
