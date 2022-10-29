package fab

import "context"

// All produces a target that runs a collection of targets in parallel.
func All(targets ...Target) Target {
	return &all{Namer: NewNamer("all"), targets: targets}
}

type all struct {
	*Namer
	targets []Target
}

var _ Target = &all{}

// Run implements Target.Run.
func (a *all) Run(ctx context.Context) error {
	return Run(ctx, a.targets...)
}

func init() {
	AddFactory("all", func(name string, conf map[string]any) (Target, error) {

		// Hm, this requires:
		//   foo: !all
		//     targets:
		//       - a
		//       - b
		//       ...etc...
		// but it would be nicer to write:
		//   foo: !all
		//     - a
		//     - b
		//     ...etc...
		// (except that doesn't allow for specifying "deps," hm).

		targets, ok := conf["targets"]
		if !ok {
			// xxx error
		}
		targetnames, ok := targets.([]string)
		if !ok {
			// xxx error
		}
		result := all{Namer: NewNamer(name)}
		for _, targetname := range targetnames {
			target, _ := RegistryTarget(targetname)
			if target == nil {
				// xxx error
			}
			result.targets = append(result.targets, target)
		}
		return &result, nil
	})
}
