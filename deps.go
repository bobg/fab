package fab

// Deps wraps a target with a set of dependencies,
// making sure those run first.
//
// It is equivalent to Seq(All(depTargets...), target).
func Deps(target Target, depTargets ...Target) Target {
	return Seq(All(depTargets...), target)
}
