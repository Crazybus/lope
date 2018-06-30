package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
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

func run(args []string, stdout bool) (output string, err error) {
	debug(fmt.Sprintf("Running: %v\n", strings.Join(args, " ")))
	cmd := exec.Command(args[0], args[1:]...)

	var out bytes.Buffer

	if stdout {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = &out
		cmd.Stderr = &out
	}
	err = cmd.Run()
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
	out, err := run(build, false)
	debug(out)
	return out, err
}

func path(p string) string {
	return filepath.FromSlash(p)
}

func cmdProxy(w http.ResponseWriter, r *http.Request) {
	type message struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}

	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var msg message
	err = json.Unmarshal(b, &msg)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	c := exec.Command(msg.Command, msg.Args...)
	out, err := c.CombinedOutput()
	if err != nil {
		fmt.Println(string(out), err)
	}

	w.Header().Set("content-type", "application/json")
	w.Write(out)
}

func getIPAddress() (ip string) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr).IP.String()
	return localAddr
}

type image struct {
	params []string
}

type config struct {
	addDocker    bool
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
	os           string
	root         bool
	sourceImage  string
	cmdProxy     bool
	cmdProxyPort string
	ssh          bool
	instructions []string
	paths        []string
	tty          bool
	workDir      string
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
		d = append(d, fmt.Sprintf("ADD . %v", l.cfg.workDir))
	}

	if l.cfg.addDocker {
		d = append(
			d,
			`RUN wget -q https://download.docker.com/linux/static/stable/x86_64/docker-18.03.1-ce.tgz && \`,
			`tar xfv docker* && \`,
			`mv docker/docker /usr/local/bin && \`,
			`rm -rf docker/`,
		)
	}

	d = append(d, l.cfg.instructions...)

	l.dockerfile = strings.Join(d, "\n")

	// If there aren't any custom instructions just use the original source image
	if len(d) == 1 {
		l.cfg.image = l.cfg.sourceImage
	}
}

// Windows has some nasty envionmental variables which aren't compatible with linux
// This strips out any non posix environment variables
func (l *lope) cleanEnvVars() {
	r, _ := regexp.Compile("^[a-zA-Z_]+[a-zA-Z0-9_]$")
	for i := len(l.envs) - 1; i >= 0; i-- {
		env := strings.Split(l.envs[i], "=")[0]
		if !r.MatchString(env) {
			l.envs = append(l.envs[:i], l.envs[i+1:]...)
		}
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
	l.params = append(
		l.params,
		"docker",
		"run",
		"--rm",
		"--interactive",
		"--entrypoint", l.cfg.entrypoint,
		"--workdir", l.cfg.workDir,
		"--net", "host",
	)
	if l.cfg.tty {
		l.params = append(
			l.params,
			"--tty",
		)
	}
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
		path := fmt.Sprintf("%v:%v", l.cfg.dir, l.cfg.workDir)
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

	if l.cfg.root {
		return
	}

	u, err := user.Current()
	// If we can't get the current user and group just ignore this since this is only a nice way
	// to avoid screwing up permissions for any files created in the bind mounted directory for
	// systems like jenkins where the default docker root user can create files that jenkins can't
	// clean up
	if err != nil {
		return
	}
	groupID := u.Gid
	// If the docker group is avaiable set it as the default group so that we can read the docker socket
	g, _ := user.LookupGroup("docker")
	if g != nil {
		groupID = g.Gid
	}
	l.params = append(l.params, fmt.Sprintf("--user=%v:%v", u.Uid, groupID))
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
	authorizedKeys, _ := run(k, false)
	authorizedKeys = base64.StdEncoding.EncodeToString([]byte(authorizedKeys))

	image := "uber/ssh-agent-forward:latest"
	name := "lope-sshd"
	volume := "lope-ssh-agent"
	port := "2244"
	host := "127.0.0.1"

	// Create a volume to mount our ssh-agent into
	r := make([]string, 0)
	r = append(r, "docker", "volume", "create", "--name", volume)
	run(r, false)

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
	run(p, false)

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
		_, err := run(w, false)
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

func (l *lope) commandProxy() {
	if !l.cfg.cmdProxy {
		return
	}

	http.HandleFunc("/", cmdProxy)
	address := ":" + l.cfg.cmdProxyPort

	debug(fmt.Sprintf("Starting lope command proxy server on address: %q", address))

	go http.ListenAndServe(address, nil)

	ip := getIPAddress()

	l.params = append(l.params, "--add-host=localhost:"+ip)

	l.envs = append(l.envs, "LOPE_PROXY_ADDR=http://"+ip+":"+l.cfg.cmdProxyPort)
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
	l.commandProxy()
	l.addVolumes()
	l.cleanEnvVars()
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
	flag.StringVar(&blacklist, "blacklist", "HOME,SSH_AUTH_SOCK,TMPDIR,PATH", "Comma seperated list of environment variables that will be ignored by lope")

	var whitelist string
	flag.StringVar(&whitelist, "whitelist", "", "Comma seperated list of environment variables that will be be included by lope")

	dir := flag.String("dir", pwd, "The directory that will be mounted into the container. Defaut is current working directory")

	entrypoint := flag.String("entrypoint", "/bin/sh", "The entrypoint for running the lope command")

	flag.Var(&instructions, "instruction", "Extra docker image instructions to run when building the image. Can be specified multiple times")

	flag.Var(&mountPaths, "path", "Paths that will be mounted from the users home directory into lope. Path will be ignored if it isn't accessible. Can be specified multiple times")

	noMount := flag.Bool("noMount", false, "Disable mounting the current working directory into the image")

	addMount := flag.Bool("addMount", false, "Setting this will add the directory into the image instead of mounting it")

	noDocker := flag.Bool("noDocker", false, "Disables mounting the docker socket inside the container")

	ssh := flag.Bool("ssh", false, "Enable forwarding ssh agent into the container")

	dockerSocket := flag.String("dockerSocket", "/var/run/docker.sock", "Path to the docker socket")

	workDir := flag.String("workDir", "/lope", "The default working directory for the docker image")

	noTty := flag.Bool("noTty", false, "Disable the --tty flag (needed for CI systems)")

	addDocker := flag.Bool("addDocker", false, "Uses wget to download the docker client binary into the image")

	noRoot := flag.Bool("noRoot", false, "Use current user instead of the root user")

	cmdProxy := flag.Bool("cmdProxy", false, "Starts a server that the lope container can use to run commands on the host")

	cmdProxyPort := flag.String("cmdProxyPort", "24242", "Listening port that will be used for the lope command proxy")

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
		addDocker:    *addDocker,
		addMount:     *addMount,
		blacklist:    strings.Split(blacklist, ","),
		cmd:          args[1:],
		cmdProxy:     *cmdProxy,
		cmdProxyPort: *cmdProxyPort,
		dir:          *dir,
		docker:       !*noDocker,
		dockerSocket: *dockerSocket,
		entrypoint:   *entrypoint,
		home:         home,
		image:        "lope",
		instructions: instructions,
		mount:        mount,
		os:           runtime.GOOS,
		paths:        paths,
		root:         !*noRoot,
		sourceImage:  args[0],
		ssh:          *ssh,
		tty:          !*noTty,
		whitelist:    strings.Split(whitelist, ","),
		workDir:      *workDir,
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

	_, err := run(params, true)
	if err != nil {
		os.Exit(1)
	}
}
