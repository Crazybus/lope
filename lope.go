package main

import (
	"bytes"
	"encoding/base64"
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
	"time"
)

func runBackground(args []string) error {
	debug(fmt.Sprintf("Starting: %v\n", strings.Join(args, " ")))
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	err := cmd.Start()
	if err != nil {
		return err
	}
	return nil
}

func run(args []string) (string, error) {
	debug(fmt.Sprintf("Running: %v\n", strings.Join(args, " ")))
	cmd := exec.Command(args[0], args[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

func buildImage(image string, dockerfile string) (string, error) {
	file, err := ioutil.TempFile(path("./"), image)
	defer os.Remove(file.Name())

	_, err = file.WriteString(dockerfile)
	if err != nil {
		log.Fatal(err)
	}

	build := make([]string, 0)
	build = append(build, "docker", "build", "-t", image, "-f", file.Name(), ".")
	out, err := run(build)
	debug(out)
	return out, err
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
	ssh          bool
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
	if l.cfg.ssh {
		l.params = append(l.params,
			"-e", "SSH_AUTH_SOCK=/ssh-agent/ssh-agent.sock",
		)
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
	if l.cfg.ssh {
		l.params = append(l.params,
			"-v", "lope-ssh-agent:/ssh-agent",
		)
	}
}

func (l *lope) addUserAndGroup() {
	// Only run the container as the host user and group if we are bind mounting the current directory
	if l.cfg.mount {
		u, err := user.Current()
		// If we can't get the current user and group just ignore this since this is only a nice way
		// to avoid screwing up permissions for any files created in the bind mounted directory for
		// systems like jenkins where the default docker root user can create files that jenkins can't
		// clean up
		if err != nil {
			return
		}
		l.params = append(l.params, fmt.Sprintf("--user=%v:%v", u.Uid, u.Gid))
	}
}

func (l *lope) runParams() {
	l.params = append(l.params, l.cfg.image, "-c", strings.Join(l.cfg.cmd, " "))
}

func (l *lope) sshForward() {

	if !l.cfg.ssh {
		return
	}

	// Get ssh keys currently added to ssh agent
	k := make([]string, 0)
	k = append(k, "ssh-add", "-L")
	authorizedKeys, _ := run(k)
	authorizedKeys = base64.StdEncoding.EncodeToString([]byte(authorizedKeys))

	image := "uber/ssh-agent-forward:latest"
	name := "lope-sshd"
	volume := "lope-ssh-agent"
	port := "2244"
	host := "127.0.0.1"

	// Create a volume to mount our ssh-agent into
	r := make([]string, 0)
	r = append(r, "docker", "volume", "create", "--name", volume)
	run(r)

	// Start the ssh server where we will forward our agent to
	p := make([]string, 0)
	p = append(
		p,
		"docker", "run",
		"--rm",
		"--name", name,
		"-e", "AUTHORIZED_KEYS="+authorizedKeys,
		"-v", volume+":/ssh-agent",
		"-d",
		"-p", port+":22",
		image,
	)
	run(p)

	// Wait for the ssh server to be responding
	w := make([]string, 0)
	w = append(
		w,
		"ssh",
		"-A",
		"-o", "StrictHostKeyChecking=no",
		"-o", "GlobalKnownHostsFile=/dev/null",
		"-o", "UserKnownHostsFile=/dev/null",
		"-p", port,
		"root@"+host,
		"ls",
	)

	for i := 1; i <= 10; i++ {
		_, err := run(w)
		if err != nil {
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}

	// Forward ssh agent into the container
	s := make([]string, 0)
	s = append(
		s,
		"ssh",
		"-A",
		"-f",
		"-o", "StrictHostKeyChecking=no",
		"-o", "GlobalKnownHostsFile=/dev/null",
		"-o", "UserKnownHostsFile=/dev/null",
		"-p", port,
		"-S", "none",
		"root@"+host,
		"/ssh-entrypoint.sh",
	)

	err := runBackground(s)
	if err != nil {
		fmt.Print("Failed to forward SSH agent", err)
	}
}

func debug(message string) {
	if _, ok := os.LookupEnv("DEBUG"); ok {
		fmt.Print("DEBUG: ", message)
	}
}

func (l *lope) run() []string {
	l.sshForward()
	l.createDockerfile()
	l.defaultParams()
	l.addVolumes()
	l.addEnvVars()
	l.addUserAndGroup()
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
	flag.StringVar(&blacklist, "blacklist", "HOME,SSH_AUTH_SOCK", "Comma seperated list of environment variables that will be ignored by lope")

	var whitelist string
	flag.StringVar(&whitelist, "whitelist", "", "Comma seperated list of environment variables that will be be included by lope")

	dir := flag.String("dir", pwd, "The directory that will be mounted into the container. Defaut is current working directory")

	entrypoint := flag.String("entrypoint", "/bin/sh", "The entrypoint for running the lope command")

	flag.Var(&instructions, "instruction", "Extra docker image instructions to run when building the image. Can be specified multiple times")

	flag.Var(&mountPaths, "path", "Paths that will be mounted from the users home directory into lope. Path will be ignored if it isn't accessible. Can be specified multiple times")

	noMount := flag.Bool("noMount", false, "Disable mounting the current working directory into the image")

	addMount := flag.Bool("addMount", false, "Setting this will add the directory into the image instead of mounting it")

	docker := flag.Bool("docker", true, "Mount the docker socket inside the container")

	noSSH := flag.Bool("noSSH", false, "Disable forwarding ssh agent into the container")

	dockerSocket := flag.String("dockerSocket", "/var/run/docker.sock", "Path to the docker socket")

	flag.Parse()
	if flag.NArg() < 2 {
		fmt.Fprintf(os.Stderr, "Usage of %[1]s:\n  %[1]s [options] <docker-image> <command>\n\nOptions:\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
		os.Exit(1)
	}
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
		ssh:          !*noSSH,
		whitelist:    strings.Split(whitelist, ","),
	}

	lope := lope{
		cfg:    config,
		envs:   os.Environ(),
		params: make([]string, 0),
	}

	params := lope.run()

	if lope.cfg.image != lope.cfg.sourceImage {
		out, err := buildImage(lope.cfg.image, lope.dockerfile)
		if err != nil {
			fmt.Println(out)
			os.Exit(1)
		}
	}

	out, err := run(params)
	fmt.Printf(out)
	if err != nil {
		os.Exit(1)
	}
}
