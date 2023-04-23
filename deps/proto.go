package deps

import (
	"bufio"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/bobg/errors"
	"github.com/bobg/go-generics/v2/set"
	"gopkg.in/yaml.v3"

	"github.com/bobg/fab"
)

// Proto reads a protocol-buffer file and returns its list of dependencies.
// Included in the dependencies is the file itself,
// plus any files it imports
// (directly or indirectly)
// that can be found among the given include directories.
// The list is sorted for consistent, predictable results.
func Proto(filename string, includes []string) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "opening %s", filename)
	}
	defer f.Close()

	result := set.New[string](filename)
	err = protodeps(f, includes, result)
	slice := result.Slice()
	sort.Strings(slice)
	return slice, err
}

var importRegex = regexp.MustCompile(`^import(\s+public)?\s*"([^"]+)"`)

func protodeps(r io.Reader, includes []string, result set.Of[string]) error {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		m := importRegex.FindStringSubmatch(sc.Text())
		if len(m) == 0 {
			continue
		}
		if err := protodepsImport(m[2], includes, result); err != nil {
			return err
		}
	}
	return sc.Err()
}

func protodepsImport(imp string, includes []string, result set.Of[string]) error {
	for _, inc := range includes {
		full := filepath.Join(inc, imp)

		if result.Has(full) {
			continue
		}

		f, err := os.Open(full)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return errors.Wrapf(err, "opening %s", full)
		}
		defer f.Close()

		result.Add(full)
		return protodeps(f, includes, result)
	}
	return nil
}

func protodepsDecoder(node *yaml.Node) ([]string, error) {
	var pd struct {
		File     string   `yaml:"File"`
		Includes []string `yaml:"Includes"`
	}
	if err := node.Decode(&pd); err != nil {
		return nil, errors.Wrap(err, "YAML error in deps.Proto node")
	}
	return Proto(pd.File, pd.Includes)
}

func init() {
	fab.RegisterYAMLStringList("deps.Proto", protodepsDecoder)
}
