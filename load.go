package fab

import (
	"context"
	"embed"
	"fmt"
	"go/ast"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"text/template"

	"github.com/fatih/camelcase"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
)

var targetMethods = make(map[string]reflect.Method)

func init() {
	var nilTarget Target
	targetType := reflect.TypeOf(nilTarget)

	for i := 0; i < targetType.NumMethod(); i++ {
		method := targetType.Method(i)
		targetMethods[method.Name] = method
	}
}

//go:embed driver.go.tmpl
var tmplFS embed.FS

func Load(ctx context.Context, pkgdir string, f func(string) error) error {
	config := &packages.Config{
		Mode:    packages.NeedName | packages.NeedTypes | packages.NeedDeps,
		Context: ctx,
	}
	pkgs, err := packages.Load(config, pkgdir)
	if err != nil {
		return errors.Wrapf(err, "loading %s", pkgdir)
	}
	if len(pkgs) != 1 {
		return errors.Wrapf(err, "found %d packages in %s, want 1", len(pkgs), pkgdir)
	}

	var (
		pkg     = pkgs[0]
		scope   = pkg.Types.Scope()
		idents  = scope.Names()
		targets []string // Top-level identifiers with types that implement fab.Target
	)
	for _, ident := range idents {
		if !ast.IsExported(ident) {
			continue
		}
		obj := scope.Lookup(ident)
		if obj == nil {
			continue
		}
		if !implementsTarget(obj.Type()) {
			continue
		}
		targets = append(targets, ident)
	}
	sort.Strings(targets)

	dir, err := os.MkdirTemp("", "fab")
	if err != nil {
		return errors.Wrap(err, "creating tempdir")
	}
	defer os.RemoveAll(dir)

	subpkgdir := filepath.Join(dir, pkg.Name)
	if err = os.Mkdir(subpkgdir, 0755); err != nil {
		return errors.Wrapf(err, "creating temp subdir %s", pkg.Name)
	}

	entries, err := os.ReadDir(pkgdir)
	if err != nil {
		return errors.Wrapf(err, "reading entries from %s", pkgdir)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		if err = copyFile(filepath.Join(pkgdir, entry.Name()), subpkgdir); err != nil {
			return errors.Wrapf(err, "copying %s to tmp subdir", entry.Name())
		}
	}

	type templateTarget struct {
		Name, SnakeName string
	}
	data := struct {
		Subpkg  string
		Targets []templateTarget
	}{
		Subpkg: pkg.Name,
	}
	for _, target := range targets {
		data.Targets = append(data.Targets, templateTarget{
			Name:      target,
			SnakeName: toSnake(target),
		})
	}

	driverOut, err := os.Create(filepath.Join(dir, "driver.go"))
	if err != nil {
		return errors.Wrap(err, "creating driver.go in temp dir")
	}
	defer driverOut.Close()

	tmpl, err := template.ParseFS(tmplFS, "driver.go.tmpl")
	if err != nil {
		return errors.Wrap(err, "parsing driver.go template")
	}
	if err = tmpl.Execute(driverOut, data); err != nil {
		return errors.Wrap(err, "rendering driver.go template")
	}
	if err = driverOut.Close(); err != nil {
		return errors.Wrap(err, "closing driver.go")
	}

	cmd := exec.CommandContext(ctx, "go", "mod", "init", "fab")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error in go mod init fab: %w; output follows\n%s", err, string(output))
	}

	cmd = exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = dir
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error in go mod tidy: %w; output follows\n%s", err, string(output))
	}

	cmd = exec.CommandContext(ctx, "go", "build")
	cmd.Dir = dir
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error in go build: %w; output follows\n%s", err, string(output))
	}

	return f(dir)
}

func implementsTarget(typ types.Type) bool {
	methodSet := types.NewMethodSet(typ)
	for name, targetMethod := range targetMethods {
		m := methodSet.Lookup(nil, name) // xxx package?
		if m == nil {
			return false
		}
		f, ok := m.Obj().(*types.Func)
		if !ok {
			return false
		}
		sig, ok := f.Type().(*types.Signature)
		if !ok {
			return false
		}
		if !signaturesMatch(sig, targetMethod.Func.Type()) {
			return false
		}
	}
	return true
}

func signaturesMatch(sig *types.Signature, fn reflect.Type) bool {
	if fn.Kind() != reflect.Func {
		return false
	}
	if sig.Variadic() != fn.IsVariadic() {
		return false
	}

	params := sig.Params()
	if params.Len() != fn.NumIn() {
		return false
	}
	results := sig.Results()
	if results.Len() != fn.NumOut() {
		return false
	}

	for i := 0; i < params.Len(); i++ {
		sp, tp := params.At(i).Type(), fn.In(i)
		if !typesMatch(sp, tp) {
			return false
		}
	}
	for i := 0; i < results.Len(); i++ {
		sr, tr := results.At(i).Type(), fn.Out(i)
		if !typesMatch(sr, tr) {
			return false
		}
	}

	return true
}

// TODO: Handle parameterized types.
func typesMatch(t types.Type, r reflect.Type) bool {
	switch t := t.(type) {
	case *types.Array:
		if r.Kind() != reflect.Array {
			return false
		}
		if t.Len() != int64(r.Len()) {
			return false
		}
		return typesMatch(t.Elem(), r.Elem())

	case *types.Basic:
		switch t.Kind() {
		case types.Bool:
			return r.Kind() == reflect.Bool
		case types.Int:
			return r.Kind() == reflect.Int
		case types.Int8:
			return r.Kind() == reflect.Int8
		case types.Int16:
			return r.Kind() == reflect.Int16
		case types.Int32:
			return r.Kind() == reflect.Int32
		case types.Int64:
			return r.Kind() == reflect.Int64
		case types.Uint:
			return r.Kind() == reflect.Uint
		case types.Uint8:
			return r.Kind() == reflect.Uint8
		case types.Uint16:
			return r.Kind() == reflect.Uint16
		case types.Uint32:
			return r.Kind() == reflect.Uint32
		case types.Uint64:
			return r.Kind() == reflect.Uint64
		case types.Uintptr:
			return r.Kind() == reflect.Uintptr
		case types.Float32:
			return r.Kind() == reflect.Float32
		case types.Float64:
			return r.Kind() == reflect.Float64
		case types.Complex64:
			return r.Kind() == reflect.Complex64
		case types.Complex128:
			return r.Kind() == reflect.Complex128
		case types.String:
			return r.Kind() == reflect.String
		case types.UnsafePointer:
			return r.Kind() == reflect.UnsafePointer
		}
		return false

	case *types.Chan:
		if r.Kind() != reflect.Chan {
			return false
		}
		switch t.Dir() {
		case types.SendRecv:
			if r.ChanDir() != reflect.BothDir {
				return false
			}
		case types.SendOnly:
			if r.ChanDir() != reflect.SendDir {
				return false
			}
		case types.RecvOnly:
			if r.ChanDir() != reflect.RecvDir {
				return false
			}
		}
		return typesMatch(t.Elem(), r.Elem())

	case *types.Interface:
		// xxx

	case *types.Map:
		if r.Kind() != reflect.Map {
			return false
		}
		if !typesMatch(t.Key(), r.Key()) {
			return false
		}
		return typesMatch(t.Elem(), r.Elem())

	case *types.Named:
		if t.Obj().Name() != r.Name() {
			return false
		}
		return typesMatch(t.Underlying(), r)

	case *types.Pointer:
		if r.Kind() != reflect.Ptr {
			return false
		}
		return typesMatch(t.Elem(), r.Elem())

	case *types.Signature:
		return signaturesMatch(t, r)

	case *types.Slice:
		if r.Kind() != reflect.Slice {
			return false
		}
		return typesMatch(t.Elem(), r.Elem())

	case *types.Struct:
		if r.Kind() != reflect.Struct {
			return false
		}
		// xxx fieldwise comparison

		// case *types.Tuple:
		// case *types.TypeParam:
		// case *types.Union:
	}

	return false
}

func copyFile(filename, destdir string) error {
	outfilename := filepath.Join(destdir, filename)
	out, err := os.OpenFile(outfilename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return errors.Wrapf(err, "creating %s", outfilename)
	}
	defer out.Close()

	in, err := os.Open(filename)
	if err != nil {
		return errors.Wrapf(err, "opening %s", filename)
	}
	defer in.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return errors.Wrapf(err, "copying %s", filename)
	}

	return out.Close()
}

func toSnake(inp string) string {
	parts := camelcase.Split(inp)
	for i := 0; i < len(parts); i++ {
		parts[i] = strings.ToLower(parts[i])
	}
	return strings.Join(parts, "_")
}
