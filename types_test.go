package fab

import (
	"go/types"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestImplementsTarget(t *testing.T) {
	t.Parallel()

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

	controllerObj := scope.Lookup("Controller")
	controllerTypeName, ok := controllerObj.(*types.TypeName)
	if !ok {
		t.Fatalf("controllerObj is a %T, want types.TypeName", controllerObj)
	}
	controllerType := controllerTypeName.Type()
	if checkImplementsTarget(controllerType) == nil {
		t.Errorf("checkImplementsTarget(fab.Controller) wrongly reports true")
	}

	commandTargetObj := scope.Lookup("Command")
	commandTargetTypeName, ok := commandTargetObj.(*types.TypeName)
	if !ok {
		t.Fatalf("commandTargetObj is a %T, want types.TypeName", commandTargetObj)
	}
	commandTargetType := commandTargetTypeName.Type()
	commandPtrType := types.NewPointer(commandTargetType)
	if err = checkImplementsTarget(commandPtrType); err != nil {
		t.Errorf("checkImplementsTarget(fab.Command) wrongly reports false: %s", err)
	}
}
