package rules

import "github.com/bobg/fab"

var Noop = fab.Shellf("echo hello")

var notExported = fab.Shellf("echo not exported")

var NotATarget = 17
