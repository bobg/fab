package loader

import (
	"context"
	_ "embed"
	"fmt"
	"go/ast"
	"go/types"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"text/template"

	"github.com/fatih/camelcase"
	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"

	"github.com/bobg/fab"
)

var targetMethods = make(map[string]reflect.Method)

// nullTarget is here so we can get reflection info about Target
type nullTarget struct{}

var _ fab.Target = nullTarget{}

func (nullTarget) ID() string                { return "" }
func (nullTarget) Run(context.Context) error { return nil }

func init() {
	var nt fab.Target = nullTarget{}
	targetType := reflect.TypeOf(nt)
	for i := 0; i < targetType.NumMethod(); i++ {
		method := targetType.Method(i)
		targetMethods[method.Name] = method
	}
}

//go:embed driver.go.tmpl
var driverStr string

func Load(ctx context.Context, pkgdir string, f func(string) error) error {
	loadpath := pkgdir
	if !filepath.IsAbs(pkgdir) {
		_, err := os.Stat(pkgdir)
		if errors.Is(err, fs.ErrNotExist) {
			// do nothing
		} else if err != nil {
			return errors.Wrapf(err, "statting %s", pkgdir)
		} else {
			loadpath = "./" + filepath.Clean(loadpath)
		}
	}
	config := &packages.Config{
		Mode:    packages.NeedName | packages.NeedTypes | packages.NeedDeps,
		Context: ctx,
	}
	pkgs, err := packages.Load(config, loadpath)
	if err != nil {
		return errors.Wrapf(err, "loading %s", loadpath)
	}
	if len(pkgs) != 1 {
		return errors.Wrapf(err, "found %d packages in %s, want 1", len(pkgs), loadpath)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		var errs []string
		for _, e := range pkg.Errors {
			errs = append(errs, e.Error())
		}
		return fmt.Errorf("error(s) loading %s: %s", loadpath, strings.Join(errs, ";\n  "))
	}

	return LoadPkg(ctx, pkgdir, pkg.Name, pkg.Types.Scope(), f)
}

func LoadPkg(ctx context.Context, pkgdir, pkgname string, scope *types.Scope, f func(string) error) error {
	var (
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
	if len(targets) == 0 {
		return fmt.Errorf("found no targets after loading %s", pkgname)
	}

	sort.Strings(targets)

	return LoadTargets(ctx, pkgdir, pkgname, targets, f)
}

func LoadTargets(ctx context.Context, pkgdir, pkgname string, targets []string, f func(string) error) error {
	dir, err := os.MkdirTemp("", "fab")
	if err != nil {
		return errors.Wrap(err, "creating tempdir")
	}
	// xxx defer os.RemoveAll(dir)

	fabsubdir := filepath.Join(dir, "fab")
	if err = os.Mkdir(fabsubdir, 0755); err != nil {
		return errors.Wrapf(err, "creating %s", fabsubdir)
	}
	goFiles, err := fab.GoFiles.ReadDir(".")
	if err != nil {
		return errors.Wrap(err, "reading GoFiles")
	}
	for _, goFile := range goFiles {
		contents, err := fab.GoFiles.ReadFile(goFile.Name())
		if err != nil {
			return errors.Wrapf(err, "reading Go file %s", goFile.Name())
		}
		dest := filepath.Join(fabsubdir, goFile.Name())
		err = os.WriteFile(dest, contents, 0644)
		if err != nil {
			return errors.Wrapf(err, "writing %s", dest)
		}
	}

	// TODO: refactor to harmonize the copying above with the copying below.

	subpkgdir := filepath.Join(dir, "pkg", pkgname)
	if err = os.MkdirAll(subpkgdir, 0755); err != nil {
		return errors.Wrapf(err, "creating %s", subpkgdir)
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
		Subpkg: pkgname,
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

	tmpl := template.New("")
	_, err = tmpl.Parse(driverStr)
	if err != nil {
		return errors.Wrap(err, "parsing driver template")
	}
	if err = tmpl.Execute(driverOut, data); err != nil {
		return errors.Wrap(err, "rendering driver.go template")
	}
	if err = driverOut.Close(); err != nil {
		return errors.Wrap(err, "closing driver.go")
	}

	cmd := exec.CommandContext(ctx, "go", "mod", "init", "x")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error in go mod init fab: %w; output follows\n%s", err, string(output))
	}

	gomodPath := filepath.Join(dir, "go.mod")
	gomodData, err := os.ReadFile(gomodPath)
	if err != nil {
		return errors.Wrapf(err, "reading %s", gomodPath)
	}
	mf, err := modfile.Parse(gomodPath, gomodData, nil)
	if err != nil {
		return errors.Wrapf(err, "parsing %s", gomodPath)
	}
	err = mf.AddReplace("github.com/bobg/fab", "", "./fab", "")
	if err != nil {
		return errors.Wrapf(err, "adding replace directive in %s", gomodPath)
	}
	gomodData, err = mf.Format()
	err = os.WriteFile(gomodPath, gomodData, 0644)
	if err != nil {
		return errors.Wrapf(err, "rewriting %s", gomodPath)
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
			fmt.Printf("  xxx no method for %s\n", name)
			return false
		}
		f, ok := m.Obj().(*types.Func)
		if !ok {
			fmt.Printf("  xxx m.Obj() is a %T, not a Func\n", m.Obj())
			return false
		}
		sig, ok := f.Type().(*types.Signature)
		if !ok {
			fmt.Printf("  xxx f.Type() is a %T, not a Signature\n", f.Type())
			return false
		}

		var comp comparer
		if !comp.signaturesMatch(sig, targetMethod.Func.Type(), true) {
			fmt.Printf("  xxx signature %s does not match %s\n", sig, targetMethod.Func.Type())
			return false
		}
	}
	return true
}

type comparer struct {
	depth int
}

func (comp *comparer) debugf(msg string, args ...any) {
	if true {
		return
	}

	fmt.Print(strings.Repeat("  ", comp.depth))
	fmt.Printf(msg, args...)
	fmt.Print("\n")
}

func (comp *comparer) signaturesMatch(sig *types.Signature, fn reflect.Type, skipReceiver bool) (result bool) {
	comp.debugf("signaturesMatch(%s, %s)", sig, fn)
	comp.depth++
	defer func() {
		comp.depth--
		comp.debugf("signaturesMatch(%s, %s) -> %v", sig, fn, result)
	}()

	if fn.Kind() != reflect.Func {
		comp.debugf("  fn.Kind() is %s, not Func", fn.Kind())
		return false
	}
	if sig.Variadic() != fn.IsVariadic() {
		comp.debugf("  sig.Variadic is %v, fn.IsVariadic is %v", sig.Variadic(), fn.IsVariadic())
		return false
	}

	hasReceiver := skipReceiver && (sig.Recv() != nil)

	params := sig.Params()
	nParamsWithReceiver := params.Len()
	if hasReceiver {
		nParamsWithReceiver++
	}

	if nParamsWithReceiver != fn.NumIn() {
		comp.debugf("hasReceiver is %v and %d does not match %d", hasReceiver, params.Len(), fn.NumIn())
		return false
	}
	results := sig.Results()
	if results.Len() != fn.NumOut() {
		comp.debugf("results.Len is %d and fn.NumOut is %d", results.Len(), fn.NumOut())
		return false
	}

	for i := 0; i < params.Len(); i++ {
		j := i
		if hasReceiver {
			j++
		}
		sp, tp := params.At(i).Type(), fn.In(j)
		if !comp.typesMatch(sp, tp) {
			return false
		}
	}
	for i := 0; i < results.Len(); i++ {
		sr, tr := results.At(i).Type(), fn.Out(i)
		if !comp.typesMatch(sr, tr) {
			return false
		}
	}

	return true
}

// TODO: Handle parameterized types.
func (comp *comparer) typesMatch(t types.Type, r reflect.Type) (result bool) {
	comp.debugf("typesMatch(%s, %s)", t, r)
	comp.depth++
	defer func() {
		comp.depth--
		comp.debugf("typesMatch(%s, %s) -> %v", t, r, result)
	}()

	switch t := t.(type) {
	case *types.Array:
		if r.Kind() != reflect.Array {
			return false
		}
		if t.Len() != int64(r.Len()) {
			return false
		}
		return comp.typesMatch(t.Elem(), r.Elem())

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
		return comp.typesMatch(t.Elem(), r.Elem())

	case *types.Interface:
		if r.Kind() != reflect.Interface {
			comp.debugf("r.Kind is %s, not Interface", r.Kind())
			return false
		}
		methodSet := types.NewMethodSet(t)
		if methodSet.Len() != r.NumMethod() {
			comp.debugf("methodSet.Len() is %d, r.NumMethod() is %d", methodSet.Len(), r.NumMethod())
			return false
		}
		for i := 0; i < methodSet.Len(); i++ {
			f, ok := methodSet.At(i).Obj().(*types.Func)
			if !ok {
				comp.debugf("methodSet.At(%d).Obj() is a %T, not a Func", i, methodSet.At(i).Obj())
				return false
			}
			method, ok := r.MethodByName(f.Name())
			if !ok {
				comp.debugf("r has no method %s", f.Name())
				return false
			}

			comp.debugf("f = %s, method.Type = %s", f, method.Type)

			sig, ok := f.Type().(*types.Signature)
			if !ok {
				comp.debugf("f.Type() is a %T, not a Signature", f.Type())
				return false
			}
			if !comp.signaturesMatch(sig, method.Type, false) {
				comp.debugf("sig %s does not match method type %s", sig, method.Type)
				return false
			}
		}
		return true

	case *types.Map:
		if r.Kind() != reflect.Map {
			return false
		}
		if !comp.typesMatch(t.Key(), r.Key()) {
			return false
		}
		return comp.typesMatch(t.Elem(), r.Elem())

	case *types.Named:
		if t.Obj().Name() != r.Name() {
			return false
		}
		return comp.typesMatch(t.Underlying(), r)

	case *types.Pointer:
		if r.Kind() != reflect.Ptr {
			return false
		}
		return comp.typesMatch(t.Elem(), r.Elem())

	case *types.Signature:
		return comp.signaturesMatch(t, r, true)

	case *types.Slice:
		if r.Kind() != reflect.Slice {
			return false
		}
		return comp.typesMatch(t.Elem(), r.Elem())

	case *types.Struct:
		if r.Kind() != reflect.Struct {
			return false
		}
		if t.NumFields() != r.NumField() {
			return false
		}
		for i := 0; i < t.NumFields(); i++ {
			v, f := t.Field(i), r.Field(i)
			if v.Name() != f.Name {
				return false
			}
			if t.Tag(i) != string(f.Tag) {
				return false
			}
			if !comp.typesMatch(v.Type(), f.Type) {
				return false
			}
		}
		return true

		// case *types.Tuple:
		// case *types.TypeParam:
		// case *types.Union:
	}

	return false
}

func copyFile(filename, destdir string) error {
	outfilename := filepath.Join(destdir, filepath.Base(filename))
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
