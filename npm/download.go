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

const MaxConcurrentDownloads = 20

type Download struct {
	module			*Module
	resultChan	chan int
}

//var downloadChannel = make(chan *Download, MaxConcurrentDownloads)

func (a *App) Install() {
	var wg sync.WaitGroup
	client := &http.Client{}

	downloadChannel := make(chan *Download, MaxConcurrentDownloads)
	quitChannel := make(chan bool)

	// can use ioutil for this instead
	tmpdir := filepath.Join(os.TempDir(), a.Name)
	os.Mkdir(tmpdir, 0755)

	go processDownloads(downloadChannel, quitChannel, client, tmpdir)

	for _, dep := range a.Dependencies {
		wg.Add(1)
		go func(module Module) {
			defer wg.Done()
			module.Install(client, downloadChannel)
		}(dep)
	}

	wg.Wait()
	quitChannel <- true

	fmt.Printf("downloaded %s to %s\n", a.Name, tmpdir)
}

// goroutine to handle incoming downloads
func processDownloads(downloads chan *Download, quit chan bool, client *http.Client, tmpdir string) {
	for range downloads {
		download := <-downloads
		go func(dl Download) {
			dl.module.Download(client, tmpdir)
			dl.resultChan <- 1
		}(*download)
	}
	<-quit
}

func (m *Module) Install(client *http.Client, downloadChannel chan *Download) {
	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()
		downloadResult := make(chan int)
		downloadChannel <- &Download{m, downloadResult}
		<-downloadResult
		fmt.Printf("done with %s\n", m.Name)
	}()

	for _, dep := range m.Dependencies {
		wg.Add(1)
		go func(module Module) {
			defer wg.Done()
			module.Install(client, downloadChannel)
			fmt.Printf("done with %s\n", module.Name)
		}(dep)
	}

	wg.Wait()
}

func (m *Module) Download(client *http.Client, directory string) {
	// fmt.Printf("Downloading %s from %s\n", m.Name, m.Resolved)

	resp, err := client.Get(m.Resolved)
	if err != nil {
		fmt.Printf("Error downloading %s from '%s'\n", m.Name, m.Resolved)
		return
	}
	defer resp.Body.Close()

	target := filepath.Join(directory, path.Base(m.Resolved))
	output, err := os.Create(target)
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()

	// fmt.Printf("Writing %s\n", target)
	io.Copy(output, resp.Body)

	/*
		if err != nil { }
	*/

	// fmt.Printf("Completed downloading %s\n", m.Name)//, resp.Status)
}
