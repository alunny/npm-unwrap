package npm

import (
	"archive/tar"
	"compress/gzip"
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
	expectedArchive := filepath.Join(tmpdir, path.Base(m.Resolved))
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

func mkPath(tarName string, baseDir string) (newPath string) {
        return strings.Replace(tarName, "package", baseDir, 1)
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
                fileDir := filepath.Dir(outputPath)

                err = os.MkdirAll(fileDir, 0755)
                if err != nil {
                        log.Fatal(err)
                }
		// fmt.Println(outputPath)

                output, err := os.Create(outputPath)
                if err != nil {
                        log.Fatal(err)
                }
                defer output.Close()

                _, err = io.Copy(output, tarReader)
                if err != nil {
                        log.Fatal(err)
                }
        }
	return
}
