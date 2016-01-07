package npm

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func (a *App) InstallFromTmpdir(tmpdir string, targetDir string) (err error) {
	err = os.MkdirAll(targetDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	for _, module := range a.Dependencies {
		err = decompressAndInstall(module, tmpdir, targetDir)
		if err != nil {
			log.Fatal(err)
		}

	}

	return
}

func decompressAndInstall(m Module, tmpdir string, targetDir string) (err error) {
	if strings.HasPrefix(m.Resolved, "git+") {
		fmt.Printf("skipping git url %s\n\t%s\n", m.Resolved, targetDir)
		return
	}

	var basePath string
	if m.Resolved == "" {
		basePath = fmt.Sprintf("%s-%s.tgz", m.Name, m.Version)
	} else {
		basePath = path.Base(m.Resolved)
	}

	expectedArchive := filepath.Join(tmpdir, basePath)
	tgz, err := os.Open(expectedArchive)
	if os.IsExist(err) {
		log.Fatalf("No file at %s\n", expectedArchive)
	}
	if err != nil {
		return err
	}
	defer tgz.Close()

	outputDir := filepath.Join(targetDir, m.Name)
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return err
	}

	err = uncompressAndExtract(tgz, outputDir)
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
			err = decompressAndInstall(module, tmpdir, nodeModulesDir)
			if err != nil {
				return err
			}
		}
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
