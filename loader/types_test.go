package loader

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
	if implementsTarget(runnerType) {
		t.Errorf("implementsTarget(fab.Runner) wrongly reports true")
	}

	commandObj := scope.Lookup("Command")
	commandTypeName, ok := commandObj.(*types.TypeName)
	if !ok {
		t.Fatalf("commandObj is a %T, want types.TypeName", commandObj)
	}
	commandType := commandTypeName.Type()
	if !implementsTarget(commandType) {
		t.Errorf("implementsTarget(fab.Command) wrongly reports false")
	}
}
