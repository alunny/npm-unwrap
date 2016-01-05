package npm

// types for dealing with NPM backed apps and modules, as serialized
// in npm-shrinkwrap.json

type Module struct {
	Name         string
	Version      string
	From         string
	Resolved     string
	Dependencies []Module
}

type App struct {
	Name         string
	Version      string
	Dependencies []Module
}

type Package interface {
	DependencyList() []Module
}

func (m Module) DependencyList() (deps []Module) {
	return m.Dependencies
}

func (a App) DependencyList() (deps []Module) {
	return a.Dependencies
}

type Download struct {
	module		*Module
	resultChan	chan int
}
