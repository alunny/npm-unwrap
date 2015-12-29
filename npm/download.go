package npm

import (
	"fmt"
	"net/http"
	"sync"
)

func (a *App) Install() {
	var wg sync.WaitGroup
	client := &http.Client{}

	for _, dep := range a.Dependencies {
		wg.Add(1)
		go func(module Module) {
			defer wg.Done()
			module.Install(client)
		}(dep)
	}

	wg.Wait()

	fmt.Printf("downloaded %s\n", a.Name)
}

func (m *Module) Install(client *http.Client) {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		m.Download(client)
	}()

	for _, dep := range m.Dependencies {
		wg.Add(1)
		go func(module Module) {
			defer wg.Done()
			module.Install(client)
		}(dep)
	}

	wg.Wait()
}

func (m *Module) Download(client *http.Client) {
	fmt.Printf("Downloading %s from %s\n", m.Name, m.Resolved)
	resp, _ := client.Get(m.Resolved)

	/*
	if err != nil { }
	*/

	fmt.Printf("Completed downloading %s with status '%s'\n", m.Name, resp.Status)
}
