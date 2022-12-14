package fab

import (
	"go/types"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestImplementsTarget(t *testing.T) {
	config := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedDeps,
	}
	pkgs, err := packages.Load(config, ".")
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

	filesTargetObj := scope.Lookup("Files")
	filesTargetTypeName, ok := filesTargetObj.(*types.TypeName)
	if !ok {
		t.Fatalf("filesTargetObj is a %T, want types.TypeName", filesTargetObj)
	}
	filesTargetType := filesTargetTypeName.Type()
	if err = checkImplementsTarget(filesTargetType); err != nil {
		t.Errorf("checkImplementsTarget(fab.Files) wrongly reports false: %s", err)
	}
}
