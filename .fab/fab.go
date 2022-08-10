package p

import "github.com/bobg/fab"

var Build = &fab.Command{
	Cmd:     "go",
	Args:    []string{"build", "./..."},
	Verbose: true,
	Prefix:  "Build",
}

var Test = &fab.Command{
	Cmd:     "go",
	Args:    []string{"test", "-race", "-cover", "./..."},
	Verbose: true,
	Prefix:  "Test",
}
