package fab_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bobg/fab"
	_ "github.com/bobg/fab/golang"
	_ "github.com/bobg/fab/proto"
	_ "github.com/bobg/fab/ts"
)

func TestBadYAML(t *testing.T) {
	t.Parallel()

	const badYAMLDir = "_testdata/badyaml"

	entries, err := os.ReadDir(badYAMLDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") {
			continue
		}
		basename := name[:len(name)-5]
		t.Run(basename, func(t *testing.T) {
			path := filepath.Join(badYAMLDir, name)
			f, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			con := fab.NewController("")
			err = con.ReadYAML(f, "yamldir")
			if err != nil {
				t.Logf("got (expected) error %s", err)
			} else {
				t.Error("got no error but wanted one")
			}
		})
	}
}
