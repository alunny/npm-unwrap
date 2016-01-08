package npm

// types for dealing with NPM backed apps and modules, as serialized
// in npm-shrinkwrap.json

import (
	"errors"
	"regexp"
)

type GitUrl struct {
	Url	string
	Ref	string
}

type PackageJSON map[string]interface{}

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

type Download struct {
	module		*Module
	resultChan	chan int
}

func (m Module) DependencyList() (deps []Module) {
	return m.Dependencies
}

func (a App) DependencyList() (deps []Module) {
	return a.Dependencies
}

// not nearly as general as npm's - see https://docs.npmjs.com/cli/install
func GitUrlFromString(str string) (gitUrl GitUrl, err error) {
	re := regexp.MustCompile("git\\+([^#]+)#?(.*)")
	groups := re.FindStringSubmatch(str)

	if len(groups) < 2 {
		err = errors.New("gitUrl: not a valid git url")
	} else if len(groups) == 1 {
		gitUrl = GitUrl{Url: groups[1], Ref: "master" }
	} else {
		gitUrl = GitUrl{Url: groups[1], Ref: groups[2] }
	}

	return
}
