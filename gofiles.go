package fab

import "embed"

//go:embed *.go go.*
var GoFiles embed.FS
