package npm

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const MaxConcurrentDownloads = 20

func (a *App) DownloadDependencies() (tmpdir string) {
	deps, gitModules, err := depsSlice(a)
	if err != nil {
		log.Fatal(err)
	}

	sort.Strings(deps)

	deps = dedupeSlice(deps)

	fmt.Printf("tarball dependencies: %d\n", len(deps))
	fmt.Printf("git dependencies: %d\n", len(gitModules))

	/*
	for _, dep := range deduped {
		fmt.Println(dep)
	}
	*/

	var moduleDir string
	useTmpDir := false

	if useTmpDir {
		moduleDir, err = ioutil.TempDir("", a.Name)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		moduleDir = ".module-cache"
		err = os.MkdirAll(moduleDir, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Printf("writing to directory: %s\n", moduleDir)

	// download all files - MaxConcurrentDownloads concurrently
	err = downloadTarballs(moduleDir, deps)
	if err != nil {
		log.Fatal(err)
	}

	err = fetchGitRepos(moduleDir, gitModules)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("downloaded dependencies to %s\n", moduleDir)

	// get all git repos

	return moduleDir
}

/*
 * takes package with tree of dependencies, returns slice of URL dependencies (tarballs)
 * and slice of git dependencies (repo URLs + refs)
 */
func depsSlice(pkg Package) (urls []string, gitModules []Module, err error) {
	npmbin, err := exec.LookPath("npm")
	if err != nil {
		log.Fatal("cannot find npm in $PATH")
	}
	npmCommand := NpmCommand{npmbin}

	for _, dep := range pkg.DependencyList() {
		if dep.Resolved == "" {
			log.Printf("[WARNING] empty resolved field for %s@%s\n", dep.Name, dep.Version)
			resolvedUrl, err := npmCommand.view(dep)
			if err != nil {
				return urls, gitModules, err
			}

			urls = append(urls, resolvedUrl)
		} else if strings.HasPrefix(dep.Resolved, "git+") {
			gitModules = append(gitModules, dep)
		} else {
			urls = append(urls, dep.Resolved)
		}

		depDeps, gitDeps, err := depsSlice(dep)
		if err != nil {
			log.Fatal(err)
			return urls, gitModules, err
		}

		urls = append(urls, depDeps...)
		gitModules = append(gitModules, gitDeps...)
	}

	return urls, gitModules, err
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

func downloadTarballs(tmpdir string, tarballs []string) (err error) {
	var wg sync.WaitGroup
	client := &http.Client{}

	downloads := make(chan string, MaxConcurrentDownloads)
	quit := make(chan bool)

	workerCount := minInt(MaxConcurrentDownloads, len(tarballs))

	wg.Add(workerCount)
	go func() {
		for i := 0; i < workerCount; i++ {
			go getTarball(i, tmpdir, downloads, client, &wg)
		}
		<-quit
	}()

	for _, tarUrl := range tarballs {
		downloads <- tarUrl
	}
	close(downloads)

	wg.Wait()
	quit <- true

	return
}

func fetchGitRepos(tmpdir string, gitModules []Module) (err error) {
	gitbin, err := exec.LookPath("git")
	if err != nil {
		log.Fatal("cannot find git in $PATH")
	}

	for _, m := range gitModules {
		gitUrl, err := GitUrlFromString(m.Resolved)
		if err != nil {
			return err
		}

		fmt.Println(gitbin)
		fmt.Printf("cloning %s from %s at ref %s\n", m.Name, gitUrl.Url, gitUrl.Ref)

		// clone
		target := fmt.Sprintf("%s__%s", m.Name, gitUrl.Ref)

		// for some reason, git needs an empty first argument - it only reads
		// the command from the second arg
		cloneArgs := []string{"", "clone", gitUrl.Url, target}
		err = execGit(gitbin, cloneArgs, tmpdir)
		if err != nil {
			return err
		}

		// checkout
		checkoutArgs := []string{"", "checkout", "-q", gitUrl.Ref}
		err = execGit(gitbin, checkoutArgs, filepath.Join(tmpdir, target))
		if err != nil {
			return err
		}
	}

	return
}

func execGit(gitbin string, args []string, wd string) (err error) {
	cmd := exec.Cmd{
		Path: gitbin,
		Args: args,
		Dir: wd,
		Stdout: ioutil.Discard,
		Stderr: os.Stderr,
	}

	err = cmd.Start()
	if err != nil {
		return
	}

	err = cmd.Wait()
	if err != nil {
		return
	}

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
	// log.Printf("done with worker %d\n", id)
	wg.Done()
	return
}

func downloadUrl(tmpdir string, tarUrl string, client *http.Client) (err error) {
	target := filepath.Join(tmpdir, path.Base(tarUrl))
	_, statErr := os.Stat(target)
	if statErr == nil {
		// log.Printf("reading %s from cache\n", path.Base(tarUrl))
		return
	}

	if !os.IsNotExist(statErr) {
		log.Println(statErr)
		return statErr
	}

	urlObj, err := url.Parse(tarUrl)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("GET", tarUrl, nil)
	if err != nil {
		log.Fatal(err)
	}

	// is this sensible? who knows!
	if strings.Contains(urlObj.Host, "artifactory") {
		//req.Header.Set("User-Agent", "alunny hates artifactory")
		req.Close = true
	}

	resp, err := client.Do(req)
	if err == io.EOF {
		err = nil
	}

	if err != nil {
		log.Printf("%q\n", err)
		log.Fatal(err)
	}
	// fmt.Printf("downloaded %s\n", url)
	defer resp.Body.Close()

	output, err := os.Create(target)
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()

	io.Copy(output, resp.Body)

	return
}
