package main

import (
	"fmt"
	"log"
	"os"

	"github.com/alunny/npm-unwrap/npm"
)

const Version = "0.0.1"

func install() {
	shrinkwrapReader, err := os.Open("./npm-shrinkwrap.json")
	if err != nil {
		log.Fatal("could not read ./npm-shrinkwrap.json")
	}
	defer shrinkwrapReader.Close()

	app, err := npm.ParseApp(shrinkwrapReader)

	if err != nil {
		log.Fatal(err)
	}

	// npm.PrintApp(app)

	app.DownloadDependencies()
	// app.Install()
}

func main() {
	if len(os.Args) > 1 {
		cmd := os.Args[1]

		if cmd == "version" {
			fmt.Printf("%s\n", Version)
			os.Exit(0)
		} else {
			log.Fatalf("unrecognized command: %s\n", cmd)
		}
	}

	install()
}
