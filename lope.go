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
	dir          string
	entrypoint   string
	blacklist    []string
	whitelist    []string
	home         string
	image        string
	mount        bool
	noMount      bool
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
	d := make([]string, 0)

	d = append(d, fmt.Sprintf("FROM %v", l.cfg.sourceImage))

	if !l.cfg.mount && !l.cfg.noMount {
		d = append(d, "ADD . /lope")
	}

	d = append(d, l.cfg.instructions...)

	l.dockerfile = strings.Join(d, "\n")
}

func (l *lope) addEnvVars() {
	for _, e := range l.envs {
		pair := strings.Split(e, "=")
		name := pair[0]
		add := true
		blacklisted := false
		for _, b := range l.cfg.blacklist {
			matched, _ := regexp.MatchString(b, name)
			if matched {
				blacklisted = true
				break
			}
		}
		if len(l.cfg.whitelist) > 0 {
			add = false
		}
		for _, w := range l.cfg.whitelist {
			matched, _ := regexp.MatchString(w, name)
			if matched {
				add = true
				break
			}
		}
		if add && !blacklisted {
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
	if l.cfg.mount {
		path := fmt.Sprintf("%v:/lope/", l.cfg.dir)
		l.params = append(l.params, "-v", path)
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

	pwd, _ := os.Getwd()

	config := &config{
		blacklist:    []string{"HOME"},
		cmd:          os.Args[2:],
		dir:          pwd,
		entrypoint:   "/bin/sh",
		home:         home,
		image:        "lope",
		instructions: []string{"RUN mkdir -p /lope && echo hellope > /lope/hellope"},
		mount:        false,
		noMount:      false,
		paths: []string{
			path(".vault-token"),
			path(".aws/"),
			path(".kube/"),
			path(".ssh/"),
		},
		sourceImage: os.Args[1],
		whitelist:   []string{"VAULT", "AWS", "GOOGLE", "GITHUB"},
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
