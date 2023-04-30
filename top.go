package fab

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bobg/errors"
	"gopkg.in/yaml.v3"
)

// TopDir finds the top directory of a project,
// given a directory inside it.
//
// The top directory is the one containing a _fab subdirectory
// or (since that might not exist)
// the one that fab.yaml files' _dir declarations are relative to.
//
// If TopDir can't find the answer in dir,
// it will look in dir's parent,
// and so on up the tree.
func TopDir(dir string) (string, error) {
	var err error
	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", errors.Wrap(err, "making relative path absolute")
	}

	// https://pkg.go.dev/os#DirFS assures us that the result of os.DirFS implements StatFS.
	return topDir(os.DirFS("/").(fs.StatFS), dir)
}

func topDir(fsys fs.StatFS, dir string) (string, error) {
	for {
		info, err := fsys.Stat(filepath.Join(dir, "_fab"))
		if err == nil && info.IsDir() {
			return dir, nil
		}
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", errors.Wrapf(err, "statting %s/_fab", dir)
		}

		result, err := topDirHelper(fsys, dir)
		if err != nil {
			return "", err
		}
		if result != "" {
			return result, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("top directory not found")
		}
		dir = parent
	}
}

func topDirHelper(fsys fs.FS, dir string) (string, error) {
	rc, err := openFabYAML(fsys, dir)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", errors.Wrapf(err, "opening YAML file in %s", dir)
	}
	defer rc.Close()

	dec := yaml.NewDecoder(rc)
	var m map[string]any
	if err = dec.Decode(&m); err != nil {
		return "", errors.Wrapf(err, "reading YAML file in %s", dir)
	}

	decl, ok := m["_dir"].(string)
	if !ok {
		return dir, nil
	}

	if filepath.IsAbs(decl) {
		return "", fmt.Errorf("absolute pathname %s in _dir decl in %s", decl, dir)
	}

	var (
		origDir  = dir
		origDecl = decl
	)

	for {
		var (
			declBase = filepath.Base(decl)
			dirBase  = filepath.Base(dir)
		)

		fmt.Printf("xxx decl %s dir %s declBase %s dirBase %s\n", decl, dir, declBase, dirBase)

		if declBase != dirBase {
			return "", fmt.Errorf("_dir decl %s does not match actual dir %s", origDecl, origDir)
		}

		dirDir := filepath.Dir(dir)
		declDir := filepath.Dir(decl)

		if dirDir == dir {
			return "", fmt.Errorf("top directory not found")
		}
		dir = dirDir

		if declDir == "." {
			return dir, nil
		}
		decl = declDir
	}
}
