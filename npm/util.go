package npm

import (
  "fmt"
  "strings"
)

func PrintApp(a App) {
  fmt.Printf("%s @ %s\n", a.Name, a.Version)
  printDependencies(a.Dependencies, 2)
}

func printDependencies(deps []Module, indent int) {
  for _, dep := range deps {
    fmt.Printf("%s %s @ %s\n", strings.Repeat("-", indent), dep.Name, dep.Version)
    printDependencies(dep.Dependencies, indent + 2)
  }
}

