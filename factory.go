package fab

type Factory func(name string, conf map[string]any) (Target, error) // xxx context param?

var (
	factoriesMu sync.Mutex // protects factories
	factories = make(map[string]Factory)
)

func AddFactory(tag string, f Factory) {
	factoriesMu.Lock()
	factories[tag] = f
	factoriesMu.Unlock()
}

func GetFactory(tag string) (Factory, bool) {
	factoriesMu.Lock()
	f, ok := factories[tag]
	factoriesMu.Unlock()
	return f, ok
}
