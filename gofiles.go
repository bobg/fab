package fab

import "embed"

// GoFiles is an embedded filesystem containing the *.go files,
// plus go.mod and go.sum,
// of the fab package.
//go:embed *.go go.*
var GoFiles embed.FS
