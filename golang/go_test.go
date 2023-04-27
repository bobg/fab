package golang

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/bobg/go-generics/v2/slices"
)

func TestGo(t *testing.T) {
	want := []string{
		"../all.go",
		"../all_test.go",
		"../argtarg.go",
		"../argtarg_test.go",
		"../clean.go",
		"../clean_test.go",
		"../command.go",
		"../compile.go",
		"../compile_test.go",
		"../context.go",
		"../context_test.go",
		"../deps.go",
		"../deps_test.go",
		"../dirhash.go",
		"../driver.go.tmpl",
		"../embeds.go",
		"../f.go",
		"../files.go",
		"../gate.go",
		"../gate_test.go",
		"../go.mod",
		"../go.sum",
		"../hash.go",
		"../hash_test.go",
		"../main.go",
		"../main_test.go",
		"../proto/deps.go",
		"../proto/deps_test.go",
		"../proto/rule.go",
		"../register.go",
		"../register_test.go",
		"../runner.go",
		"../runner_test.go",
		"../seq.go",
		"../seq_test.go",
		"../sqlite/db.go",
		"../sqlite/db_test.go",
		"../sqlite/migrations/20220829093104_initial.sql",
		"../target.go",
		"../ts/tsdecls.go",
		"../types.go",
		"../types_test.go",
		"../yaml.go",
		"../yaml_test.go",
		"go.go",
		"go_test.go",
	}

	got, err := Deps(".", false)
	if err != nil {
		t.Fatal(err)
	}

	// Use relative paths to make the result non-system-dependent.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	got, err = slices.Mapx(got, func(_ int, full string) (string, error) {
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