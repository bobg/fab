# Fab - software fabricator

This is fab, a system for orchestrating software builds in Go.

Like Make,
fab executes recipes to turn inputs into outputs,
making sure to build prerequisites before the targets that depend on them,
and avoiding recompilation of targets that don’t need it.

Unlike Make, recipes are written in Go.
(Which is not to say that what you’re building has to be in Go; it doesn’t.)

## How it works

You create a package of Go code in your project.
By default the `fab` program looks for the package in the `fab.d` subdir.
Every exported symbol in that package
whose type satisfies the `fab.Target` interface
is a target that fab can run.

## Under the hood

When you run `fab`,
it compiles a new Go binary using your package of build rules and a custom `main` function.

## Why not Mage?

Fab was strongly inspired by [Mage](https://magefile.org/),
which has a similar feature set.
However, the author found Mage a little cumbersome for a couple of particular use cases:

- Adding persistent hashes of targets
  to determine when running one can be skipped,
  because its outputs are already up to date
  with respect to its inputs.
- Propagating errors outward from within target implementations.
- Defining targets as the result of suitably typed expressions assigned to top-level vars,
  instead of having to be Go `func`s.
