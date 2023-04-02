package deps

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/bobg/go-generics/slices"
)

func TestGo(t *testing.T) {
	want := []string{
		"go.go",
		"proto.go",
	}

	got, err := Go(".", false)
	if err != nil {
		t.Fatal(err)
	}

	// Use relative paths to make the result non-system-dependent.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	got, err = slices.Map(got, func(_ int, full string) (string, error) {
		return filepath.Rel(cwd, full)
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(got)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
