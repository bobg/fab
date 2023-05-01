# Fab - software fabricator

[![Go Reference](https://pkg.go.dev/badge/github.com/bobg/fab.svg)](https://pkg.go.dev/github.com/bobg/fab)
[![Go Report Card](https://goreportcard.com/badge/github.com/bobg/fab)](https://goreportcard.com/report/github.com/bobg/fab)
[![Tests](https://github.com/bobg/fab/actions/workflows/go.yml/badge.svg)](https://github.com/bobg/fab/actions/workflows/go.yml)
[![Coverage Status](https://coveralls.io/repos/github/bobg/fab/badge.svg?branch=main)](https://coveralls.io/github/bobg/fab?branch=main)

This is Fab,
a system for orchestrating software builds.
It’s like [Make](https://en.wikipedia.org/wiki/Make_(software)),
except that you express build rules and dependencies as [Go](https://go.dev/) code.

(But that doesn’t mean it’s for building Go programs only,
any more than writing shell commands in a Makefile means Make builds only shell programs.)

Running `fab` on one or more targets ensures that the targets’ prerequisites,
and the targets themselves,
are up to date according to your build rules,
while avoiding unnecessarily rebuilding any target that is already up to date.

## Usage

You will need an installation of Go version 1.20 or later.
Download Go here: [go.dev/dl](https://go.dev/dl/).

Once Go is installed you can install Fab like this:

```sh
go install github.com/bobg/fab/cmd/fab@latest
```

To define build targets in your software project,
write Go code in a `_fab` subdirectory
and/or write a `fab.yaml` file.
See [Targets](#Targets) below for how to do this.

To build targets in your software project,
run

```sh
fab TARGET1 TARGET2 ...
```

To see the progress of your build you can add the `-v` flag (for “verbose”):

```sh
fab -v TARGET1 TARGET2 ...
```

If you have a target that takes command-line parameters,
you can invoke it like this:

```sh
fab TARGET ARG1 ARG2 ...
```

In this form,
`ARG1` must start with a `-`,
and no other targets may be specified.

To see the available build targets in your project,
run

```sh
fab -list
```

## Targets

Each fab target has a _type_
that dictates the parameters required by the target (if any),
and the actions that the target should take when it runs.

Fab predefines several target types.
Here is a partial list:

- `Command` invokes a shell command when it runs.
- `F` invokes an arbitrary Go function.
- `Files` specifies a set of input files and a set of output files,
  and a nested subtarget that runs
  only when the outputs are out-of-date with respect to the inputs.
- `All` invokes a set of subtargets in parallel.
- `Seq` invokes subtargets in sequence.
- `Deps` invokes a subtarget only after its dependencies
  (other subtargets)
  have run.
- `Clean` deletes a set of files.

You define targets by instantiating one of these types,
supplying it with any necessary arguments,
and giving it a name.
There are three ways to do this:
statically in Go code;
dynamically in Go code;
and declaratively in a YAML file.
These options are discussed below.

You can also define new target types
by implementing the [fab.Target](https://pkg.go.dev/github.com/bobg/fab#Target) interface.

## Static target definition in Go

You can write Go code to define targets that Fab can run.
To do this,
create a subdirectory named `_fab` at the root of your project,
and create `.go` files in that directory.
This name prevents the code in that directory
from being considered as part of the public API for your project.
([Citation](https://pkg.go.dev/go/build#Context.Import).)

You can use any package name for the code in directory `_fab`;
the official suggestion is `_fab`,
to match the directory name.

Any exported identifiers at the top level of this package
whose type implements the [fab.Target](https://pkg.go.dev/github.com/bobg/fab#Target) interface
are usable Fab targets.
For example:

```go
package _fab

import (
    "os"

    "github.com/bobg/fab"
)

// Test runs tests.
var Test = &fab.Command{Shell: "go test -cover ./...", Stdout: os.Stdout}
```

This creates a `Command`-typed target named `Test`,
which you can invoke with `fab Test`.
When it runs,
it executes the given shell command
and copies its output to fab’s standard output.

The comment above the variable declaration gets attached to the created target.
Running `fab -list` will show that comment as a docstring:

```sh
$ fab -list
Test
    Test runs tests.
```

## Dynamic target definition in Go

Not all targets are suitable for creation via top-level variable declarations.
Those that require more complex processing can be defined dynamically
using the [fab.RegisterTarget](https://pkg.go.dev/github.com/bobg/fab#RegisterTarget) function.

```go
for m := time.January; m <= time.December; m++ {
  fab.RegisterTarget(
    m.String(),
    "Say that it’s "+m.String(),
    &fab.Command{Shell: "echo It is "+m.String(), Stdout: os.Stdout},
  )
}
```

This creates targets named `January`, `February`, `March`, etc.

Internally, static target definition works by calling `fab.RegisterTarget`.

## Declarative target definition in YAML

In addition to Go code in the `_fab` subdirectory,
or instead of it,
you can define targets in a `fab.yaml` file
at the top level of your project.

The top-level structure of the YAML file is a mapping from names to targets.
Targets are specified using YAML type tags.
Most Fab target types define a tag and a syntax for extracting necessary arguments from YAML.

Targets may also be referred to by name.

Here is an example `fab.yaml` file:

```yaml
# Prog rebuilds prog if any of the files it depends on changes.
Prog: !Files
  In: !go.Deps
    Dir: cmd/prog
  Out:
    - prog
  Target: Build

# Unconditionally rebuild prog.
Build: !Command
  Shell: go build -o prog ./cmd/prog

# Test runs all tests.
Test: !Command
  Shell: go test -race -cover ./...
  Stdout: $stdout
```

This defines `Prog` as a Fab target of type `Files`.
The `In` argument is the list of input files that may change
and the `Out` argument is the list of expected output files that result from running the nested subtarget.
That subtarget is a reference to the `Build` rule,
which is a `Command` that runs `go build`.
`In` is defined as the result of the [go.Deps](https://pkg.go.dev/github.com/bobg/fab/golang#Deps) rule,
which produces the list of files on which the Go package in a given directory depends.

This also defines a `Test` target as a `Command` that runs `go test`.

All of the target types in the `github.com/bobg/fab` package are available to your YAML file by default.
To make other target types available,
it is necessary to import their packages
in Go code in the `_fab` directory.
For example,
to make the `!go.Binary` tag work,
you’ll need a `.go` file under `_fab` that contains:

```go
import _ "github.com/bobg/fab/golang"
```

(The `_` means your Go code doesn’t use anything in the `golang` package directly,
but imports it for its side effects —
namely, registering YAML tags like `!go.Binary`.)

If you rely entirely on YAML files,
it’s possible that your `.go` code will contain only `import` statements like this
and not define any targets or types,
which is fine.

## Defining new target types

You can define new target types in Go code in the `_fab` subdirectory
(or anywhere else, that is then imported into the `_fab` package).

Your type must implement [fab.Target](https://pkg.go.dev/github.com/bobg/fab#Target),
which requires two methods: `Desc` and `Run`.

`Desc` produces a short string describing the target.
It is used by [Describe](https://pkg.go.dev/github.com/bobg/fab#Describe)
to describe targets that don’t have a name
(i.e., ones that were never registered with `RegisterTarget`,
possibly because they are nested inside some other target).

`Run` should unconditionally execute your target type’s logic.
The Fab runtime will take care of making sure your target runs only when it needs to.
More about this appears below.

If part of your `Run` method involves running other targets,
do not invoke their `Run` methods directly.
Instead, invoke the `Run` method on the [Controller](https://pkg.go.dev/github.com/bobg/fab#Controller)
that your `Run` method receives as an argument,
passing it the target you want to run.
This will skip that target if it has already run.
This means that a “diamond dependency” —
A depends on B and C,
and B and C each separately depend on X —
won’t cause X to run twice when the user runs `fab A`.

Your implementation should be a pointer type,
which is required for targets passed to [Describe](https://pkg.go.dev/github.com/bobg/fab#Describe)
and [RegisterTarget](https://pkg.go.dev/github.com/bobg/fab#RegisterTarget).

If you would like your type to be usable as the subtarget in a [Files](https://pkg.go.dev/github.com/bobg/fab#Files) rule,
it must be JSON-encodable (unlike [F](https://pkg.go.dev/github.com/bobg/fab#F), for example).
Among other things,
this means that struct fields should be [exported](https://go.dev/ref/spec#Exported_identifiers),
or it should implement [json.Marshaler](https://pkg.go.dev/encoding/json#Marshaler).
See [json.Marshal](https://pkg.go.dev/encoding/json#Marshal) for more detail on what’s encodable.

If you would like your new target type to be usable in `fab.yaml`,
you must define a YAML parser for it.
This is done with [RegisterYAMLTarget](https://pkg.go.dev/github.com/bobg/fab#RegisterYAMLTarget),
which associates a `name` with a [YAMLTargetFunc](https://pkg.go.dev/github.com/bobg/fab#YAMLTargetFunc).
When the YAML tag `!name` is encountered in `fab.yaml`
(in a context where a target may be specified),
your function will be invoked to parse the YAML node.
The function [YAMLTarget](https://pkg.go.dev/github.com/bobg/fab#YAMLTarget) parses a YAML node into a Target
using the functions in this registry.

There is also a registry for functions that parse a YAML node into a list of strings.
For example, this YAML snippet:

```yaml
!go.Deps
  Dir: foo/bar
  Recursive: true
```

produces the list of files on which the Go code in `foo/bar` depends.
You can add functions to _this_ registry with [RegisterYAMLStringList](https://pkg.go.dev/github.com/bobg/fab#RegisterYAMLStringList),
and parse a YAML node into a string list using functions from this registry with [YAMLStringList](https://pkg.go.dev/github.com/bobg/fab#YAMLStringList).

## Files

The [Files](https://pkg.go.dev/github.com/bobg/fab#Files) target type
specifies a set of input files,
a set of expected output files,
and a nested subtarget for producing one from the other.
It uses this information in two special ways:
for _file chaining_
and for _content-based dependency checking._

### File chaining

When a `Files` target runs,
it looks for filenames in its input list
that appear in the output lists of other `Files` targets.
Other targets found in this way are [Run](https://pkg.go.dev/github.com/bobg/fab#Run) first as prerequisites.

Here is a simple example in YAML:

```yaml
AB: !Files
  In: [a]
  Out: [b]
  Target: !Command
    Shell: cp a b

BC: !Files
  In: [b]
  Out: [c]
  Target: !Command
    Shell: cp b c
```

(File a produces b by copying; file b produces c by copying.)

If you run `fab BC` to update c from b,
Fab will discover that the input file b
appears in the output list of target `AB`,
and run that target first.

(If b is already up to date with respect to a,
running `AB` will have no effect.
See the next section for more about this.)

### Content-based dependency checking

After running any prerequisites found via file chaining,
a `Files` target computes a _hash_ combining the content of all the input files,
all the output files (those that exist),
and the rules for the nested subtarget.
It then checks for the presence of this hash in a persistent _hash database_
that records the state of things after any successful past run of the target.

If the hash is there, then the run succeeds trivially;
the output files are already up to date with respect to the inputs,
and running the subtarget is skipped.

Otherwise the nested subtarget runs,
and then the hash is computed again and placed into the hash database.
The next time this target runs,
if none of the files has changed,
then the hash will be the same
and running the subtarget will be skipped.
On the other hand,
if any file has changed,
the hash will be different and won’t be found in the database,
so the subtarget will run.

(It is possible for input and output files to change
in such a way that the hash _is_ found in the database,
because they match a previous “up to date” state.
Consider a simple `Files` rule for example
that copies a single input file `in`
to a single output file `out`.
Let’s say the first time it runs,
`in` contains `Hello` and that gets copied to `out`,
and the resulting post-run hash is 17.
[Actual hashes are much, much, _much_ bigger numbers.]
Now you change `in` to contain `Goodbye` and rerun the target.
The hash with `in=Goodbye` and `out=Hello` isn’t in the database,
so the copy rule runs again
and the new hash is 42.
If you now change both `in` _and_ `out` back to `Hello`
and rerun the target,
the hash will again be 17,
representing an earlier state where `out` is up to date
with respect to `in`,
so there is no copying needed.)

This is a key difference between Fab and Make.
Make uses file modification times
to decide when a set of output files needs to be recomputed from their inputs.
Considering the limited resolution of filesystem timestamps,
the possibility of clock skew, etc.,
the content-based test that Fab uses is preferable.
(But it would be easy to define a file-modtime-based target type in Fab
if that’s what you wanted.)

The hash database is stored in `$HOME/.cache/fab` by default,
and hash values normally expire after thirty days.

### Using the Files target type to translate Makefiles

It is possible to translate Makefile rules to Fab rules using the `Files` target type.

The following Makefile snippet means,
“produce files a and b from input files c and d
by running command1 followed by command2.”

```Makefile
a b: c d
  command1
  command2
```

The same thing in Fab’s YAML format looks like this.

```yaml
Name: !Files
  - In: [c, d]
  - Out: [a, b]
  - Target: !Seq
    - !Command
      Shell: command1
    - !Command
      Shell: command2
```

Note that the Fab version has a `Name` whereas the Make version does not.

## The Fab runtime

A Fab [Controller](https://pkg.go.dev/github.com/bobg/fab#Controller)
is responsible for invoking targets’ `Run` methods,
keeping track of which ones have already run
so that they don’t get invoked a second time.

The controller uses the address of each target as a unique key.
This means that pointer types should be used to implement `Target`.
After a target runs,
the controller records its outcome
(error or no error).
The second and subsequent attempts to run a given target
will use the previously computed outcome.

## Program startup

If you have Go code in a `_fab` subdirectory,
Fab combines it with its own `main` function to produce a _driver,_
which is an executable Go binary that is stored in `$HOME/.cache/fab` by default.

The driver calls [fab.RegisterTarget](https://pkg.go.dev/github.com/bobg/fab#RegisterTarget)
on each of the eligible top-level identifiers in your package;
then it looks for a `fab.yaml` file and registers the target definitions it finds there.
After that,
the driver runs the targets you specified on the `fab` command line
(or lists targets if you specified `-list`, etc).

When you run `fab` and the driver is already built and up to date
(as determined by a _hash_ of the code in the `_fab` dir),
then `fab` simply executes the driver without rebuilding it.
You can force a rebuild of the driver by specifying `-f` to `fab`.

If you do not have a `_fab` subdirectory,
then Fab operates in “driverless” mode,
in which the `fab.yaml` file is loaded
and the targets on the command line executed.

Note that when you have both a `_fab` subdirectory and a `fab.yaml` file,
you may use target types in the YAML file that are defined in your `_fab` package.
When you have only a `fab.yaml` file you are limited to the target types that are predefined in Fab.

## Why not Mage?

Fab was strongly inspired by the excellent [Mage](https://magefile.org/) tool,
which works similarly and has a similar feature set.
But Fab has some features the author needed and did not find in Mage:

- Errors from `Target` rules propagate out instead of causing an exit.
- Targets are values, not functions, and are composable (e.g. with `Seq`, `All`, and `Deps`).
- Rebuilding of up-to-date targets can be skipped based on file contents, not modtimes.
