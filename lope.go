package main

import (
	"bytes"
	"flag"
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

func run(args []string) string {
	debug(fmt.Sprintf("Running: %v\n", strings.Join(args, " ")))
	cmd := exec.Command(args[0], args[1:]...)
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
	build = append(build, "docker", "build", "-t", image, "-f", file.Name(), ".")
	debug(fmt.Sprintf(run(build)))
}

func path(p string) string {
	return filepath.FromSlash(p)
}

type image struct {
	params []string
}

type config struct {
	addMount     bool
	cmd          []string
	dir          string
	docker       bool
	dockerSocket string
	entrypoint   string
	blacklist    []string
	whitelist    []string
	home         string
	image        string
	mount        bool
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

	if l.cfg.addMount {
		d = append(d, "ADD . /lope")
	}

	d = append(d, l.cfg.instructions...)

	l.dockerfile = strings.Join(d, "\n")

	// If there aren't any custom instructions just use the original source image
	if len(d) == 1 {
		l.cfg.image = l.cfg.sourceImage
	}
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
	l.params = append(l.params, "docker", "run", "--rm", "--entrypoint", l.cfg.entrypoint, "-w", "/lope")
}

func (l *lope) addVolumes() {
	for _, p := range l.cfg.paths {
		absPath := l.cfg.home + p
		if _, err := os.Stat(absPath); err == nil {
			volume := fmt.Sprintf("%v:/root/%v", absPath, p)
			debug(fmt.Sprintf("Adding volume %q\n", volume))
			l.params = append(l.params, "-v", volume)
		}
	}
	if l.cfg.mount {
		path := fmt.Sprintf("%v:/lope/", l.cfg.dir)
		l.params = append(l.params, "-v", path)
	}
	if l.cfg.docker {
		l.params = append(l.params, "-v", l.cfg.dockerSocket+":/var/run/docker.sock")
	}
}

func (l *lope) runParams() {
	l.params = append(l.params, l.cfg.image, "-c", strings.Join(l.cfg.cmd, " "))
}

func debug(message string) {
	if _, ok := os.LookupEnv("DEBUG"); ok {
		fmt.Print("DEBUG: ", message)
	}
}

func (l *lope) run() []string {
	l.createDockerfile()
	l.defaultParams()
	l.addVolumes()
	l.addEnvVars()
	l.runParams()
	return l.params
}

type flagArray []string

func (i *flagArray) String() string {
	return "my string representation"
}

func (i *flagArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var instructions flagArray
var mountPaths flagArray

func main() {

	user, _ := user.Current()
	home := user.HomeDir + string(os.PathSeparator)

	pwd, _ := os.Getwd()

	var blacklist string
	flag.StringVar(&blacklist, "blacklist", "HOME", "Comma seperated list of environment variables that will be ignored by lope")

	var whitelist string
	flag.StringVar(&whitelist, "whitelist", "", "Comma seperated list of environment variables that will be be included by lope")

	dir := flag.String("dir", pwd, "The directory that will be mounted into the container. Defaut is current working directory")

	entrypoint := flag.String("entrypoint", "/bin/sh", "The entrypoint for running the lope command")

	flag.Var(&instructions, "instruction", "Extra docker image instructions to run when building the image. Can be specified multiple times")

	flag.Var(&mountPaths, "path", "Paths that will be mounted from the users home directory into lope. Path will be ignored if it isn't accessible. Can be specified multiple times")

	noMount := flag.Bool("noMount", false, "Disable mounting the current working directory into the image")

	addMount := flag.Bool("addMount", false, "Setting this will add the directory into the image instead of mounting it")

	docker := flag.Bool("docker", true, "Mount the docker socket inside the container")

	dockerSocket := flag.String("dockerSocket", "/var/run/docker.sock", "Path to the docker socket")

	flag.Parse()
	args := flag.Args()

	mount := !*addMount && !*noMount

	paths := []string{}
	if len(mountPaths) == 0 {
		paths = []string{
			path(".vault-token"),
			path(".aws/"),
			path(".kube/"),
			path(".ssh/"),
		}
	} else {
		for _, p := range mountPaths {
			paths = append(paths, path(p))
		}
	}

	config := &config{
		addMount:     *addMount,
		blacklist:    strings.Split(blacklist, ","),
		cmd:          args[1:],
		dir:          *dir,
		docker:       *docker,
		dockerSocket: *dockerSocket,
		entrypoint:   *entrypoint,
		home:         home,
		image:        "lope",
		instructions: instructions,
		mount:        mount,
		paths:        paths,
		sourceImage:  args[0],
		whitelist:    strings.Split(whitelist, ","),
	}

	lope := lope{
		cfg:    config,
		envs:   os.Environ(),
		params: make([]string, 0),
	}

	params := lope.run()

	if lope.cfg.image != lope.cfg.sourceImage {
		buildImage(lope.cfg.image, lope.dockerfile)
	}

	fmt.Printf(run(params))
}
