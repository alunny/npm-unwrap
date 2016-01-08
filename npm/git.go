package npm

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func copyGitModule(m Module, tmpdir string, target string) (err error) {
	gitUrl, err := GitUrlFromString(m.Resolved)
	if err != nil {
		return
	}

	sourceDir := filepath.Join(tmpdir, fmt.Sprintf("%s__%s", m.Name, gitUrl.Ref))

	err = filepath.Walk(sourceDir, func (path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return nil
		}

		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		if !info.IsDir() && info.Name() == ".gitignore" {
			return nil
		}
		if path == sourceDir {
			return nil
		}

		relativePath := strings.TrimPrefix(path, sourceDir)
		targetPath := filepath.Join(target, relativePath)

		fmt.Printf("%s, mode: %t, move to %s\n", relativePath, info.Mode(), targetPath)
		if info.IsDir() {
			err = os.MkdirAll(targetPath, info.Mode())
			if err != nil {
				return err
			}
			fmt.Println("created new directory")
		} else {
			output, err := os.OpenFile(targetPath, os.O_CREATE | os.O_RDWR, info.Mode())
			if err != nil {
				return err
			}
			defer output.Close()

			input, err := os.Open(path)
			if err != nil {
				return err
			}
			defer input.Close()

			_, err = io.Copy(output, input)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return
}

