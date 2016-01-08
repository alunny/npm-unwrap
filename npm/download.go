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

	tmpdir, err = ioutil.TempDir("", a.Name)
	if err != nil {
		log.Fatal(err)
	}

	// download all files - MaxConcurrentDownloads concurrently
	err = downloadTarballs(tmpdir, deps)
	if err != nil {
		log.Fatal(err)
	}

	err = fetchGitRepos(tmpdir, gitModules)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("downloaded dependencies to %s\n", tmpdir)

	// get all git repos

	return tmpdir
}

/*
 * takes package with tree of dependencies, returns slice of URL dependencies (tarballs)
 * and slice of git dependencies (repo URLs + refs)
 */
func depsSlice(pkg Package) (urls []string, gitModules []Module, error error) {
	for _, dep := range pkg.DependencyList() {
		if dep.Resolved == "" {
			// TODO: potentially resolve with npm-view
			fmt.Printf("[WARNING] empty resolved field for %s@%s\n", dep.Name, dep.Version)
			continue
		}

		if strings.HasPrefix(dep.Resolved, "git+") {
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

	return urls, gitModules, error
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
	log.Printf("done with worker %d\n", id)
	wg.Done()
	return
}

func downloadUrl(tmpdir string, tarUrl string, client *http.Client) (err error) {
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

	target := filepath.Join(tmpdir, path.Base(tarUrl))
	output, err := os.Create(target)
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()

	io.Copy(output, resp.Body)

	return
}
