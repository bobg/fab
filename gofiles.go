package fab

import "embed"

//go:embed *.go
var GoFiles embed.FS
