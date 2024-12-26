package fab

import (
	"golang.org/x/tools/go/packages"
)

// LoadMode is the minimal set of flags to enable for Config.Mode in a call to Packages.Load
// in order to produce a suitable package object for CompilePackage.
const LoadMode = packages.NeedName | packages.NeedFiles | packages.NeedTypes | packages.NeedDeps
