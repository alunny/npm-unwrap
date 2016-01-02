package npm

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
)

func (a *App) Install() {
	var wg sync.WaitGroup
	client := &http.Client{}
	// can use ioutil for this instead
	tmpdir := filepath.Join(os.TempDir(), a.Name)
	os.Mkdir(tmpdir, 0755)

	for _, dep := range a.Dependencies {
		wg.Add(1)
		go func(module Module) {
			defer wg.Done()
			module.Install(client, tmpdir)
		}(dep)
	}

	wg.Wait()

	fmt.Printf("downloaded %s to %s\n", a.Name, tmpdir)
}

func (m *Module) Install(client *http.Client, directory string) {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		m.Download(client, directory)
	}()

	for _, dep := range m.Dependencies {
		wg.Add(1)
		go func(module Module) {
			defer wg.Done()
			module.Install(client, directory)
		}(dep)
	}

	wg.Wait()
}

func (m *Module) Download(client *http.Client, directory string) {
	fmt.Printf("Downloading %s from %s\n", m.Name, m.Resolved)
	resp, _ := client.Get(m.Resolved)
	defer resp.Body.Close()

	target := filepath.Join(directory, path.Base(m.Resolved))
	output, err := os.Create(target)
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()

	fmt.Printf("Writing %s\n", target)
	io.Copy(output, resp.Body)

	/*
		if err != nil { }
	*/

	fmt.Printf("Completed downloading %s with status '%s'\n", m.Name, resp.Status)
}
