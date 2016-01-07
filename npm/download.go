package npm

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const MaxConcurrentDownloads = 20

func (a *App) DownloadDependencies() {
	deps, gitDeps, err := depsSlice(a)
	if err != nil {
		log.Fatal(err)
	}

	sort.Strings(deps)
	sort.Strings(gitDeps)

	deps = dedupeSlice(deps)
	gitDeps = dedupeSlice(gitDeps)

	fmt.Printf("tarball dependencies: %d\n", len(deps))
	fmt.Printf("git dependencies: %d\n", len(gitDeps))

	/*
	for _, dep := range deduped {
		fmt.Println(dep)
	}
	*/

	// download all files - MaxConcurrentDownloads concurrently
	tmpdir, err := downloadTarballs(a.Name, deps)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("downloaded dependencies to %s\n", tmpdir)

	// get all git repos
}

/*
 * takes package with tree of dependencies, returns slice of URL dependencies (tarballs)
 * and slice of git dependencies (repo URLs + refs)
 */
func depsSlice(pkg Package) (urls []string, gitUrls []string, error error) {
	for _, dep := range pkg.DependencyList() {
		if dep.Resolved == "" {
			// TODO: potentially resolve with npm-view
			fmt.Printf("[WARNING] empty resolved field for %s@%s\n", dep.Name, dep.Version)
			continue
		}

		if strings.HasPrefix(dep.Resolved, "git+") {
			fmt.Printf("[WARNING] git url for %s\n%s\n", dep.Name, dep.Resolved)
			gitUrls = append(gitUrls, dep.Resolved)
		} else {
			urls = append(urls, dep.Resolved)
		}

		depDeps, gitDeps, err := depsSlice(dep)
		if err != nil {
			return urls, gitUrls, err
		}

		urls = append(urls, depDeps...)
		gitUrls = append(gitUrls, gitDeps...)
	}

	return urls, gitUrls, error
}

// takes sorted slice orig, returns deduped (still sorted) slice
func dedupeSlice(orig []string) (deduped []string) {
	deduped = make([]string, 0, len(orig))

	for i := 0; i < len(orig); i++ {
		if len(deduped) == 0 || deduped[len(deduped) - 1] != orig[i] {
			deduped = append(deduped, orig[i])
		}
	}

	return deduped
}

func minInt(a int, b int) (min int) {
	if a < b {
		return a
	} else {
		return b
	}
}

func downloadTarballs(appName string, tarballs []string) (tmpdir string, err error) {
	var wg sync.WaitGroup
	client := &http.Client{}
	tmpdir, err = ioutil.TempDir("", appName)
	if err != nil {
		log.Fatal(err)
	}

	downloads := make(chan string, MaxConcurrentDownloads)
	quit := make(chan bool)

	workerCount := minInt(MaxConcurrentDownloads, len(tarballs))
	workerCount = 1

	wg.Add(workerCount)
	go func() {
		for i := 0; i < workerCount; i++ {
			go getTarball(i, tmpdir, downloads, client, &wg)
		}
		<-quit
	}()

	for _, url := range tarballs {
		downloads <- url
	}
	close(downloads)

	wg.Wait()
	quit <- true

	return
}

func getTarball(id int, tmpdir string, downloads chan string, client *http.Client, wg *sync.WaitGroup) (err error) {
	for dl := range downloads {
		err = downloadUrl(tmpdir, dl, client)
		if err != nil {
			fmt.Printf("Error downloading %s\n", dl)
			log.Fatal(err)
		}
	}
	wg.Done()
	return
}

func downloadUrl(tmpdir string, url string, client *http.Client) (error error) {
	resp, err := client.Get(url)
	if err != nil {
		return
	}
	// fmt.Printf("downloaded %s\n", url)
	defer resp.Body.Close()

	target := filepath.Join(tmpdir, path.Base(url))
	output, err := os.Create(target)
	if err != nil {
		return
	}
	defer output.Close()

	io.Copy(output, resp.Body)

	return
}
