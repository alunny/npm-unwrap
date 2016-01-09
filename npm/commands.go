package npm

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"strings"
)

type NpmCommand struct {
	BinPath	string
}

// shells out to `npm view` to get tarball URL
func (npmCmd NpmCommand) view(m Module) (url string, err error) {
	var out bytes.Buffer
	viewArgs := []string{"npm", "view", m.Name + "@" + m.Version, "dist.tarball"}

	cmd := exec.Cmd{
		Path: npmCmd.BinPath,
		Args: viewArgs,
		Stdout: &out,
		Stderr: os.Stderr,
	}

	err = cmd.Start()
	if err != nil {
		log.Println(err)
		return
	}

	err = cmd.Wait()
	if err != nil {
		log.Println(out.String())
		log.Println(err)
		return
	}

	url = strings.TrimSpace(out.String())
	log.Printf("URL: %s\n", url)

	return
}
