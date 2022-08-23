package rules

import "github.com/bobg/fab"

var Noop = &fab.Command{Shell: "sh -c 'echo hello'"}
