package rules

import "github.com/bobg/fab"

var Noop = fab.Command("sh -c 'echo hello'")

var notExported = fab.Command("sh -c 'echo not exported'")

var NotATarget = 17
