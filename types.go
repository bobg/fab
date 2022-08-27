package fab

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"
)

var targetInterface *types.Interface

func init() {
	tmpdir, err := os.MkdirTemp("", "fab")
	if err != nil {
		log.Fatal(err)
	}
	// xxx defer os.RemoveAll(tmpdir)

	for _, filename := range []string{"target.go", "go.mod", "go.sum"} {
		contents, err := embeds.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(tmpdir, filename), contents, 0644); err != nil {
			log.Fatal(err)
		}
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, tmpdir, nil, 0)
	if err != nil {
		log.Fatal(err)
	}
	if len(pkgs) != 1 {
		log.Fatalf("Got %d packages, want 1", len(pkgs))
	}
	var astpkg *ast.Package
	for _, p := range pkgs {
		astpkg = p
		break
	}

	var files []*ast.File
	for _, f := range astpkg.Files {
		files = append(files, f)
	}

	conf := types.Config{Importer: importer.Default()}
	tpkg, err := conf.Check("github.com/bobg/fab", fset, files, nil)
	if err != nil {
		log.Fatal(err)
	}

	ttype := tpkg.Scope().Lookup("Target").Type()
	var ok bool
	targetInterface, ok = ttype.Underlying().(*types.Interface)
	if !ok {
		log.Fatalf("Type of Target is %T, not *types.Interface", ttype)
	}

	fmt.Printf("xxx targetInterface is %s %#v\n", targetInterface, targetInterface)
}

func implementsTarget(typ types.Type) bool {
	return types.Implements(typ, targetInterface)
}
