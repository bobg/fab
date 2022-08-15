package p

import "github.com/bobg/fab"

var Build = &fab.Command{
	Cmd:    "go",
	Args:   []string{"build", "./..."},
	Prefix: "Build",
}

var Test = &fab.Command{
	Cmd:     "go",
	Args:    []string{"test", "-race", "-cover", "./..."},
	Prefix:  "Test",
	Verbose: true,
}

var Lint = &fab.Command{
	Cmd:     "staticcheck",
	Args:    []string{"./..."},
	Prefix:  "Lint",
	Verbose: true,
}
