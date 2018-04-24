package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
)

// https://npf.io/2015/06/testing-exec-command/

func dockerRun(args []string) string {
	fmt.Printf("Running: docker %v\n", strings.Join(args, " "))
	cmd := exec.Command("docker", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(out.String())
	}
	return out.String()
}

func buildImage(image string, dockerfile string) {
	file, err := ioutil.TempFile(path("./"), image)
	defer os.Remove(file.Name())

	_, err = file.WriteString(dockerfile)
	if err != nil {
		log.Fatal(err)
	}

	build := make([]string, 0)
	build = append(build, "build", "-t", image, "-f", file.Name(), ".")
	fmt.Println(dockerRun(build))
}

func path(p string) string {
	return filepath.FromSlash(p)
}

type image struct {
	params []string
}

type config struct {
	cmd          []string
	entrypoint   string
	envBlacklist []string
	envPattern   string
	home         string
	image        string
	sourceImage  string
	instructions []string
	paths        []string
}

type lope struct {
	cfg        *config
	dockerfile string
	envs       []string
	params     []string
}

func (l *lope) createDockerfile() {
	l.dockerfile = fmt.Sprintf("FROM %v\nADD . /lope\n", l.cfg.sourceImage)
	l.dockerfile = l.dockerfile + strings.Join(l.cfg.instructions, "\n")
}

func (l *lope) addEnvVars() {
	for _, e := range l.envs {
		pair := strings.Split(e, "=")
		blacklisted := false
		for _, b := range l.cfg.envBlacklist {
			if pair[0] == b {
				blacklisted = true
				break
			}
		}
		if l.cfg.envPattern != "" {
			matched, _ := regexp.MatchString(l.cfg.envPattern, e)

			if !matched {
				blacklisted = true
			}
		}

		if !blacklisted {
			l.params = append(l.params, "-e", e)
		}
	}
}

func (l *lope) defaultParams() {
	l.params = append(l.params, "run", "--rm", "--entrypoint", l.cfg.entrypoint, "-w", "/lope")
}

func (l *lope) addVolumes() {
	for _, p := range l.cfg.paths {
		absPath := l.cfg.home + p
		if _, err := os.Stat(absPath); err == nil {
			volume := fmt.Sprintf("%v:/root/%v", absPath, p)
			fmt.Printf("Adding volume %q\n", volume)
			l.params = append(l.params, "-v", volume)
		}
	}
}

func (l *lope) runParams() {
	l.params = append(l.params, l.cfg.image, "-c", strings.Join(l.cfg.cmd, " "))
}

func (l *lope) run() []string {
	l.createDockerfile()
	l.defaultParams()
	l.addVolumes()
	l.addEnvVars()
	l.runParams()
	return l.params
}

func main() {

	user, _ := user.Current()
	home := user.HomeDir + string(os.PathSeparator)

	config := &config{
		cmd:          os.Args[2:],
		entrypoint:   "/bin/sh",
		envBlacklist: []string{"HOME"},
		envPattern:   "VAULT|AWS|GOOGLE|GITHUB",
		home:         home,
		sourceImage:  os.Args[1],
		image:        "lope",
		instructions: []string{"RUN echo hellope > /lope/hellope"},
		paths: []string{
			path(".vault-token"),
			path(".aws/"),
			path(".kube/"),
			path(".ssh/"),
		},
	}

	lope := lope{
		cfg:    config,
		envs:   os.Environ(),
		params: make([]string, 0),
	}

	params := lope.run()
	buildImage(lope.cfg.image, lope.dockerfile)
	fmt.Println(dockerRun(params))
}
