package main

import (
	"log"
	"os"

	"github.com/alunny/npm-unwrap/npm"
)

func main() {
	shrinkwrapReader, err := os.Open("./npm-shrinkwrap.json")
	if err != nil {
		log.Fatal("could not read ./npm-shrinkwrap.json")
	}
	defer shrinkwrapReader.Close()

	app, err := npm.ParseApp(shrinkwrapReader)

	if err != nil {
		log.Fatal(err)
	}

	npm.PrintApp(app)

	app.Install()
}
