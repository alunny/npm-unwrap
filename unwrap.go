package main

import (
  "fmt"
  "os"
  "log"

  "github.com/alunny/npm-unwrap/npm"
)

func main() {
  shrinkwrapReader, err := os.Open("./npm-shrinkwrap.json")

  if err != nil {
    log.Fatal("could not read ./npm-shrinkwrap.json")
  }

  app, err := npm.ParseApp(shrinkwrapReader)

  if err != nil {
    log.Fatal(err)
  }

  fmt.Printf("%s @ %s\n", app.Name, app.Version)
}
