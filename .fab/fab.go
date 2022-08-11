package p

import "github.com/bobg/fab"

var Build = &fab.Command{
	Cmd:    "go",
	Args:   []string{"build", "./..."},
	Prefix: "Build",
}

var Test = &fab.Command{
	Cmd:    "go",
	Args:   []string{"test", "-race", "-cover", "./..."},
	Prefix: "Test",
}

var Lint = &fab.Command{
	Cmd:    "revive",
	Args:   []string{"./..."},
	Prefix: "Lint",
	Verbose: true,
}
