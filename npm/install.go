package npm

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

func (a *App) InstallFromTmpdir(tmpdir string, targetDir string) (err error) {
	npmbin, err := exec.LookPath("npm")
	if err != nil {
		log.Fatal("cannot find npm in $PATH")
	}

	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	for _, module := range a.Dependencies {
		err = installModule(module, tmpdir, targetDir, npmbin)
		if err != nil {
			return err
		}
	}

	return
}

func installModule(m Module, tmpdir string, targetDir string, npmbin string) (err error) {
	var isGitModule bool
	if strings.HasPrefix(m.Resolved, "git+") {
		isGitModule = true
	}

	outputDir := filepath.Join(targetDir, m.Name)
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return err
	}

	if isGitModule {
		err = copyGitModule(m, tmpdir, outputDir)
	} else {
		err = decompress(m, tmpdir, outputDir)
	}

	if err != nil {
		return err
	}

	if len(m.Dependencies) > 0 {
		nodeModulesDir := filepath.Join(outputDir, "node_modules")
		err = os.MkdirAll(nodeModulesDir, 0755)
		if err != nil {
			return err
		}

		for _, module := range m.Dependencies {
			err = installModule(module, tmpdir, nodeModulesDir, npmbin)
			if err != nil {
				return err
			}
		}
	}

	pkg, err := ReadPackageJSON(outputDir)
	if err != nil {
		return err
	}

	err = pkg.runInstallScripts(npmbin, outputDir)
	if err != nil {
		return err
	}

	err = pkg.linkBinScripts(npmbin, outputDir)
	if err != nil {
		return err
	}

	return
}

func decompress(m Module, tmpdir string, outputDir string) (err error) {
	var basePath string
	if m.Resolved == "" {
		basePath = fmt.Sprintf("%s-%s.tgz", m.Name, m.Version)
	} else {
		basePath = path.Base(m.Resolved)
	}

	expectedArchive := filepath.Join(tmpdir, basePath)

	tgz, err := os.Open(expectedArchive)
	if os.IsNotExist(err) {
		log.Fatalf("No file at %s\n", expectedArchive)
	}
	if err != nil {
		return err
	}
	defer tgz.Close()

	err = uncompressAndExtract(tgz, outputDir)
	if err != nil {
		return err
	}

	return
}

func (pkg PackageJSON) linkBinScripts(npmbin string, directory string) (err error) {
	binScripts, err := pkg.BinScripts()
	if err != nil {
		return err
	}

	if len(binScripts) == 0 {
		return
	}

	binDir := filepath.Join(directory, "../.bin")
	err = os.MkdirAll(binDir, 0755)
	if err != nil {
		return err
	}

	for scriptName, scriptPath := range binScripts {
		source := filepath.Join(directory, scriptPath)
		target := filepath.Join(binDir, scriptName)

		err = os.Symlink(source, target)
		if err != nil {
			return err
		}
	}

	return
}

func (pkg PackageJSON) runInstallScripts(npmbin string, directory string) (err error) {
	runInstall := []string{"run-script", "install", "--production"}

	hasInstall, err := pkg.HasInstallScript()
	if err != nil {
		return
	}

	// check for binding.gyp
	var hasBindingGyp bool
	_, err = os.Stat(filepath.Join(directory, "binding.gyp"))
	if os.IsNotExist(err) {
		err = nil
		hasBindingGyp = false
	} else if err != nil {
		return
	} else {
		hasBindingGyp = true
	}

	if hasInstall || hasBindingGyp {
		fmt.Printf("run '%s install' for %s\n", npmbin, directory)
		cmd := exec.Cmd{
			Path: npmbin,
			Args: runInstall,
			Dir: directory,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}
		err = cmd.Start()
		if err != nil {
			return
		}

		cmd.Wait()
	}

	return
}

func mkPath(entry string, baseDir string) (newPath string) {
	segments := strings.SplitAfterN(entry, string(os.PathSeparator), 2)
	if len(segments) != 2 {
		return ""
	} else {
		return filepath.Join(baseDir, segments[1])
	}
}

/*
 based on: https://github.com/npm/npm/blob/2.x/lib/utils/tar.js
 equivalent to
 	gzip {tarball} --decompress --stdout | tar -mvxpf - --strip-components=1 -C {unpackTarget}
*/
func uncompressAndExtract(tgz *os.File, outputDir string) (err error) {
	decompressor, err := gzip.NewReader(tgz)
        if err != nil {
                return err
        }
        defer decompressor.Close()

        tarReader := tar.NewReader(decompressor)
        for {
                header, err := tarReader.Next()
                if err == io.EOF {
                        break
                }
                if err != nil {
                        return err
                }

		info := header.FileInfo()
		if info.IsDir() {
			continue
		}


                outputPath := mkPath(header.Name, outputDir)
		if outputPath == "" {
			fmt.Printf("[WARN] invalid entry %s\n", header.Name)
			continue
		}

                fileDir := filepath.Dir(outputPath)
                err = os.MkdirAll(fileDir, 0755)
                if err != nil {
                        return err
                }
		// fmt.Println(outputPath)

		err = writeFile(outputPath, tarReader)
                if err != nil {
                        return err
                }
        }
	return
}

// move to separate function to ensure files are correctly closed in spite of errors
func writeFile(path string, reader io.Reader) (err error) {
	// TODO: preserve file permissions
	output, err := os.Create(path)
	if err != nil {
		return err
	}
	defer output.Close()

	_, err = io.Copy(output, reader)
	if err != nil {
		return err
	}

	return
}
