package internal

import (
	"go/types"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestImplementsTarget(t *testing.T) {
	config := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedDeps,
	}
	pkgs, err := packages.Load(config, "..")
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages, want 1", len(pkgs))
	}
	scope := pkgs[0].Types.Scope()

	runnerObj := scope.Lookup("Runner")
	runnerTypeName, ok := runnerObj.(*types.TypeName)
	if !ok {
		t.Fatalf("runnerObj is a %T, want types.TypeName", runnerObj)
	}
	runnerType := runnerTypeName.Type()
	if checkImplementsTarget(runnerType) == nil {
		t.Errorf("checkImplementsTarget(fab.Runner) wrongly reports true")
	}

	filesCommandObj := scope.Lookup("FilesCommand")
	filesCommandTypeName, ok := filesCommandObj.(*types.TypeName)
	if !ok {
		t.Fatalf("filesCommandObj is a %T, want types.TypeName", filesCommandObj)
	}
	filesCommandType := filesCommandTypeName.Type()
	if err = checkImplementsTarget(filesCommandType); err != nil {
		t.Errorf("checkImplementsTarget(fab.FilesCommand) wrongly reports false: %s", err)
	}
}
