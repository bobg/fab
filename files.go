package fab

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v2/maps"
	"github.com/bobg/go-generics/v2/slices"
	json "github.com/gibson042/canonicaljson-go"
	"gopkg.in/yaml.v3"
)

var filesRegistry = newRegistry[*files]()

// Files creates a target that contains a list of input files
// and a list of expected output files.
// It also contains a nested subtarget
// whose Run method should produce or update the expected output files.
//
// When the Files target runs,
// it does the following:
//
//   - It checks to see whether any of its input files
//     are listed as output files in other Files targets.
//     Other targets found in this way are run first,
//     as prerequisites.
//   - It then computes a hash from the nested subtarget
//     and all the input and output files.
//     If this hash is found in the “hash database”
//     (obtained with [GetHashDB]),
//     that means none of the files has changed
//     since the last time the output files were built,
//     so running of the subtarget can be skipped.
//   - Otherwise the subtarget is run.
//     The hash is then recomputed
//     and added to the hash database,
//     telling the next run of this target
//     that this collection of input and output files
//     can be considered up-to-date.
//
// The nested subtarget must be of a type that can be JSON-marshaled.
// Notably this excludes [F].
//
// The list of input files should mention every file where a change should cause a rebuild.
// Ideally this includes any files required by the nested subtarget
// plus any transitive dependencies.
// See the Deps function in the golang subpackage
// for an example of a function that can compute such a list for a Go package.
//
// Passing Autoclean(true) as one of the options
// causes the output files to be added to the "autoclean registry."
// A [Clean] target may then choose to remove the files listed in that registry
// (instead of, or in addition to, any explicitly listed files)
// by setting _its_ Autoclean field to true.
//
// The list of input and output files may include directories too.
// These are walked recursively for computing the hash described above.
// Be careful when using directories in the output-file list
// together with the Autoclean feature:
// the entire directory tree will be deleted.
//
// When [GetDryRun] is true,
// checking and updating of the hash DB is skipped.
//
// A Files target may be specified in YAML using the !Files tag,
// which introduces a mapping whose fields are:
//
//   - Target: the nested subtarget, or target name
//   - In: the list of input files, interpreted with [YAMLFilesList]
//   - Out: the list of output files, interpreted with [YAMLFilesList]
//   - Autoclean: a boolean
//
// Example:
//
//	Foo: !Files
//	  Target: !Command
//	    - go build -o thingify ./cmd/thingify
//	  In: !golang.Deps
//	    Dir: cmd
//	  Out:
//	    - thingify
//
// This creates target Foo,
// which runs the given `go build` command
// to update the output file `thingify`
// when any files depended on by the Go package in `cmd` change.
func Files(target Target, in, out []string, opts ...FilesOpt) Target {
	result := &files{
		Target: target,
		In:     in,
		Out:    out,
	}

	for _, opt := range opts {
		opt(result)
	}

	for _, o := range out {
		filesRegistry.add(o, result)
	}

	return result
}

type files struct {
	Target Target
	In     []string
	Out    []string
}

var _ Target = &files{}

// Run implements Target.Run.
func (ft *files) Run(ctx context.Context, con *Controller) error {
	if err := ft.runPrereqs(ctx, con); err != nil {
		return errors.Wrap(err, "in prerequisites")
	}

	db := GetHashDB(ctx)

	if db != nil && !GetForce(ctx) && !GetDryRun(ctx) {
		h, err := ft.computeHash(con)
		if err != nil {
			return errors.Wrap(err, "computing hash before running subtarget")
		}
		has, err := db.Has(ctx, h)
		if err != nil {
			return errors.Wrap(err, "checking hash db")
		}
		if has {
			if GetVerbose(ctx) {
				con.Indentf("%s is up to date", con.Describe(ft))
			}
			return nil
		}
	}

	if err := con.Run(ctx, ft.Target); err != nil {
		return errors.Wrap(err, "running subtarget")
	}

	if db == nil || GetDryRun(ctx) {
		return nil
	}

	h, err := ft.computeHash(con)
	if err != nil {
		return errors.Wrap(err, "computing hash after running subtarget")
	}
	err = db.Add(ctx, h)
	return errors.Wrap(err, "adding hash to db")
}

// Desc implements Target.Desc.
func (*files) Desc() string {
	return "Files"
}

func (ft *files) computeHash(con *Controller) ([]byte, error) {
	inHashes, err := fileHashes(ft.In)
	if err != nil {
		return nil, errors.Wrapf(err, "computing input hash(es) for %s", con.Describe(ft))
	}
	outHashes, err := fileHashes(ft.Out)
	if err != nil {
		return nil, errors.Wrapf(err, "computing output hash(es) for %s", con.Describe(ft))
	}
	tt := reflect.TypeOf(ft.Target)
	s := struct {
		Target     Target   `json:"target"`
		TargetType string   `json:"target_type"`
		In         []string `json:"in,omitempty"`  // [filename, hash, filename, hash, ...]
		Out        []string `json:"out,omitempty"` // [filename, hash, filename, hash, ...]
	}{
		Target:     ft.Target,
		TargetType: tt.String(),
		In:         inHashes,
		Out:        outHashes,
	}
	j, err := json.Marshal(s)
	if err != nil {
		return nil, errors.Wrap(err, "in JSON marshaling")
	}

	sum := sha256.Sum224(j)
	return sum[:], nil
}

func (ft *files) runPrereqs(ctx context.Context, con *Controller) error {
	var prereqs []Target

	for _, in := range ft.In {
		if target := findInFilesRegistry(in); target != nil {
			prereqs = append(prereqs, target)
		}
	}

	if len(prereqs) == 0 {
		return nil
	}
	return con.Run(ctx, prereqs...)
}

func findInFilesRegistry(name string) Target {
	for {
		if target, ok := filesRegistry.lookup(name); ok {
			return target
		}

		dir := filepath.Dir(name)
		switch dir {
		case "", ".", "/", name:
			return nil
		}
		name = dir
	}
}

type FilesOpt func(*files)

// Autoclean is an option for passing to [Files].
// It causes the output files of the Files target to be added to the "autoclean registry."
// A [Clean] target may then choose to remove the files listed in that registry
// (instead of, or in addition to, any explicitly listed files)
// by setting its Autoclean field to true.
func Autoclean(autoclean bool) FilesOpt {
	return func(f *files) {
		if !autoclean {
			return
		}
		autocleanMu.Lock()
		for _, file := range f.Out {
			autocleanRegistry.Add(file)
		}
		autocleanMu.Unlock()
	}
}

// Returns [filename, hash, filename, hash, ...],
// with filenames sorted.
// Input is a list of file or directory names.
func fileHashes(items []string) ([]string, error) {
	hashes := make(map[string]string)

	if err := fileHashesHelper(items, hashes); err != nil {
		return nil, err
	}

	keys := maps.Keys(hashes)
	sort.Strings(keys)

	result := make([]string, 0, 2*len(keys))
	for _, key := range keys {
		result = append(result, key, hashes[key])
	}

	return result, nil
}

func fileHashesHelper(items []string, hashes map[string]string) error {
	for _, item := range items {
		if err := fileHashesItemHelper(item, hashes); err != nil {
			return err
		}
	}

	return nil
}

func fileHashesItemHelper(item string, hashes map[string]string) error {
	if _, ok := hashes[item]; ok {
		// Already computed.
		// (There can be duplicates or overlaps in the input.)
		return nil
	}

	info, err := os.Stat(item)
	if errors.Is(err, fs.ErrNotExist) {
		hashes[item] = ""
		return nil
	}

	if info.IsDir() {
		entries, err := os.ReadDir(item)
		if err != nil {
			return errors.Wrapf(err, "reading directory %s", item)
		}
		subitems := slices.Map(entries, func(s os.DirEntry) string { return filepath.Join(item, s.Name()) })
		return fileHashesHelper(subitems, hashes)
	}

	h, err := hashFile(item)
	if err != nil {
		return errors.Wrapf(err, "hashing file %s", item)
	}
	hashes[item] = h

	return nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", errors.Wrapf(err, "opening %s", path)
	}
	defer f.Close()
	hasher := sha256.New224()
	_, err = io.Copy(hasher, f)
	if err != nil {
		return "", errors.Wrapf(err, "hashing %s", path)
	}
	h := hasher.Sum(nil)
	return hex.EncodeToString(h), nil
}

func filesDecoder(con *Controller, node *yaml.Node, dir string) (Target, error) {
	if node.Kind != yaml.MappingNode {
		return nil, BadYAMLNodeKindError{Got: node.Kind, Want: yaml.MappingNode}
	}

	var yfiles struct {
		In        yaml.Node `yaml:"In"`
		Out       yaml.Node `yaml:"Out"`
		Target    yaml.Node `yaml:"Target"`
		Autoclean bool      `yaml:"Autoclean"`
	}
	if err := node.Decode(&yfiles); err != nil {
		return nil, errors.Wrap(err, "YAML error in Files node")
	}

	target, err := con.YAMLTarget(&yfiles.Target, dir)
	if err != nil {
		return nil, errors.Wrap(err, "YAML error in Target child of Files node")
	}

	in, err := con.YAMLFileList(&yfiles.In, dir)
	if err != nil {
		return nil, errors.Wrap(err, "YAML error in Files.In node")
	}

	out, err := con.YAMLFileList(&yfiles.Out, dir)
	if err != nil {
		return nil, errors.Wrap(err, "YAML error in Files.Out node")
	}

	return Files(target, in, out, Autoclean(yfiles.Autoclean)), nil
}

func globDecoder(node *yaml.Node) ([]string, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, BadYAMLNodeKindError{Got: node.Kind, Want: yaml.SequenceNode}
	}

	patterns, err := YAMLStringListFromNodes(node.Content)
	if err != nil {
		return nil, errors.Wrap(err, "in children of Glob node")
	}

	var result []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, errors.Wrap(err, "in Glob pattern")
		}
		result = append(result, matches...)
	}

	return result, nil
}

func init() {
	RegisterYAMLTarget("Files", filesDecoder)
	RegisterYAMLStringList("Glob", globDecoder)
}
