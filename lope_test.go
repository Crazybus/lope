package main

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
)

var c = &config{
	cmd:          []string{"ls", "-lhatr"},
	dockerSocket: "/var/run/docker.sock",
	entrypoint:   "/bin/sh",
	blacklist:    []string{""},
	whitelist:    []string{""},
	home:         "/home/lope",
	image:        "lopeImage",
	instructions: []string{""},
	workDir:      "/lope",
	paths: []string{
		path(".vault-token"),
		path(".aws/"),
		path(".kube/"),
		path(".ssh/"),
	},
}

var l = lope{
	cfg:        c,
	dockerfile: "imageName",
	envs: []string{
		"VAULT_ADDR=http://localhost:8200",
		"VAULT_TOKEN=123456",
	},
	params: make([]string, 0),
}

func TestRunParams(t *testing.T) {

	var tests = []struct {
		description string
		cmd         []string
		image       string
		want        string
	}{
		{
			"Run command with image and cmd",
			[]string{"command"},
			"imageName",
			"imageName -c command",
		},
		{
			"Run command with image and multiple cmd args",
			[]string{"command", "-arg"},
			"imageName",
			"imageName -c command -arg",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			l.params = make([]string, 0)
			l.cfg.cmd = test.cmd
			l.cfg.image = test.image
			l.runParams()

			got := strings.Join(l.params, " ")
			want := test.want

			if got != want {
				t.Errorf("got %q want %q", got, want)
			}
		})
	}
}

func TestAddVolumes(t *testing.T) {

	var tests = []struct {
		description string
		paths       []string
		home        string
		mount       bool
		docker      bool
		ssh         bool
		dir         string
		want        string
	}{
		{
			"Add the aws directory",
			[]string{".aws"},
			path("./test/"),
			false,
			false,
			false,
			"",
			fmt.Sprintf("-v %v.aws:/root/.aws", path("./test/")),
		},
		{
			"Don't add any directories if they don't exist",
			[]string{".aws", ".not-exist"},
			path("./test/"),
			false,
			false,
			false,
			"",
			fmt.Sprintf("-v %v.aws:/root/.aws", path("./test/")),
		},
		{
			"Don't add any directories if none are defined",
			[]string{},
			path("./test/"),
			false,
			false,
			false,
			"",
			"",
		},
		{
			"Mount the specified directory if -mount is set",
			[]string{},
			path("./test/"),
			true,
			false,
			false,
			"/home/user/pro/lope/",
			"-v /home/user/pro/lope/:/lope",
		},
		{
			"Add multiple directories",
			[]string{".aws", ".kube"},
			path("./test/"),
			false,
			false,
			false,
			"",
			fmt.Sprintf("-v %v.aws:/root/.aws -v %v.kube:/root/.kube", path("./test/"), path("./test/")),
		},
		{
			"Mount the docker socket",
			[]string{},
			path("./test/"),
			false,
			true,
			false,
			"",
			"-v /var/run/docker.sock:/var/run/docker.sock",
		},
		{
			"Mount ssh volumes if ssh is enabled",
			[]string{},
			path("./test/"),
			false,
			false,
			true,
			"",
			"-v lope-ssh-agent:/ssh-agent",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			l.params = make([]string, 0)
			l.cfg.home = test.home
			l.cfg.paths = test.paths
			l.cfg.mount = test.mount
			l.cfg.dir = test.dir
			l.cfg.docker = test.docker
			l.cfg.ssh = test.ssh
			l.addVolumes()

			got := strings.Join(l.params, " ")
			want := test.want

			if got != want {
				t.Errorf("got %q want %q", got, want)
			}
		})
	}
}

func TestAddEnvVars(t *testing.T) {

	var tests = []struct {
		description string
		envs        []string
		blacklist   []string
		whitelist   []string
		ssh         bool
		want        string
	}{
		{
			"Add an env var",
			[]string{"ENV1=hello1"},
			[]string{},
			[]string{},
			false,
			"-e ENV1",
		},
		{
			"Add multiple env vars",
			[]string{"ENV1=hello1", "ENV2=hello2"},
			[]string{},
			[]string{},
			false,
			"-e ENV1 -e ENV2",
		},
		{
			"Blacklist an env var",
			[]string{"ENV1=hello1"},
			[]string{"ENV1"},
			[]string{},
			false,
			"",
		},
		{
			"Whitelist an env var",
			[]string{"ENV1=hello1", "ENV2=hello2", "NO=no"},
			[]string{},
			[]string{"ENV"},
			false,
			"-e ENV1 -e ENV2",
		},
		{
			"Blacklist and whitelisting env vars",
			[]string{"ENV1=hello1", "ENV2=hello2", "NO=no"},
			[]string{"ENV1"},
			[]string{"ENV"},
			false,
			"-e ENV2",
		},
		{
			"Add the SSH auth socket if ssh is enabled",
			[]string{},
			[]string{},
			[]string{},
			true,
			"-e SSH_AUTH_SOCK=/ssh-agent/ssh-agent.sock",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			l.params = make([]string, 0)
			l.envs = test.envs
			l.cfg.blacklist = test.blacklist
			l.cfg.whitelist = test.whitelist
			l.cfg.ssh = test.ssh
			l.addEnvVars()

			got := strings.Join(l.params, " ")
			want := test.want

			if got != want {
				t.Errorf("got %q want %q", got, want)
			}
		})
	}
}

func TestDefaultParams(t *testing.T) {

	var tests = []struct {
		description string
		entrypoint  string
		tty         bool
		want        string
	}{
		{
			"Override the entrypoint",
			"/bin/ohyeah",
			false,
			"docker run --rm --interactive --entrypoint /bin/ohyeah --workdir /lope --net host",
		},
		{
			"Allocate a pseudo-TTY",
			"/bin/ohyeah",
			true,
			"docker run --rm --interactive --entrypoint /bin/ohyeah --workdir /lope --net host --tty",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			l.params = make([]string, 0)
			l.cfg.entrypoint = test.entrypoint
			l.cfg.tty = test.tty
			l.defaultParams()

			got := strings.Join(l.params, " ")
			want := test.want

			if got != want {
				t.Errorf("got %q want %q", got, want)
			}
		})
	}
}

func TestCreateDockerfile(t *testing.T) {

	var tests = []struct {
		description  string
		image        string
		mount        bool
		addMount     bool
		addDocker    bool
		instructions []string
		want         []string
	}{
		{
			"Create a basic dockerfile",
			"imageName",
			false,
			true,
			false,
			[]string{""},
			[]string{
				"FROM imageName",
				"ADD . /lope",
				"",
			},
		},
		{
			"Dont ADD the files if mount is set",
			"imageName",
			true,
			false,
			false,
			[]string{""},
			[]string{
				"FROM imageName",
				"",
			},
		},
		{
			"Dont ADD the files if addMount is not set",
			"imageName",
			false,
			false,
			false,
			[]string{""},
			[]string{
				"FROM imageName",
				"",
			},
		},
		{
			"Create a dockerfile with extra instructions",
			"imageName",
			false,
			false,
			false,
			[]string{
				"RUN echo hello",
				"RUN hello world",
			},
			[]string{
				"FROM imageName",
				"RUN echo hello",
				"RUN hello world",
			},
		},
		{
			"Add docker binary to image",
			"imageName",
			false,
			false,
			true,
			[]string{},
			[]string{
				"FROM imageName",
				`RUN wget -q https://download.docker.com/linux/static/stable/x86_64/docker-18.03.1-ce.tgz && \`,
				`tar xfv docker* && \`,
				`mv docker/docker /usr/local/bin && \`,
				`rm -rf docker/`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			l.cfg.sourceImage = test.image
			l.cfg.instructions = test.instructions
			l.cfg.mount = test.mount
			l.cfg.addMount = test.addMount
			l.cfg.addDocker = test.addDocker
			l.createDockerfile()

			got := l.dockerfile
			want := strings.Join(test.want, "\n")

			if got != want {
				t.Errorf("got %q want %q", got, want)
			}
		})
	}
}
func TestUseSourceImage(t *testing.T) {
	l.cfg.sourceImage = "canttouchthis"
	l.cfg.addMount = false
	l.cfg.instructions = []string{}
	l.cfg.addDocker = false
	l.createDockerfile()

	got := l.cfg.image
	want := l.cfg.sourceImage

	if l.cfg.image != l.cfg.sourceImage {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestUserAndGroupParams(t *testing.T) {

	var tests = []struct {
		description string
		mount       bool
		os          string
		want        string
	}{
		{
			"--users is NOT set if mount is false",
			false,
			"",
			"",
		},
		{
			"--users is NOT set if mount is true but os isn't linux",
			true,
			"windows",
			"",
		},
		{
			"--users IS set if mount is true",
			true,
			"linux",
			fmt.Sprintf("--user="),
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			l.params = make([]string, 0)
			l.cfg.mount = test.mount
			l.cfg.os = test.os
			l.addUserAndGroup()

			got := strings.Join(l.params, " ")
			want := test.want

			if !strings.HasPrefix(got, want) {
				t.Errorf("got %q wanted prefix: %q", got, want)
			}
		})
	}
}

func TestCleanEnvVars(t *testing.T) {

	var tests = []struct {
		description string
		envs        []string
		want        string
	}{
		{
			"All env vars are valid",
			[]string{
				"TEST=hello",
			},
			"TEST=hello",
		},
		{
			"Invalid env vars are stripped",
			[]string{
				"TEST=hello",
				"T:EST=hello",
			},
			"TEST=hello",
		},
		{
			"Only the key name is checked for invalid characters",
			[]string{
				"TEST=he:llo",
			},
			"TEST=he:llo",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			l.envs = test.envs
			l.cleanEnvVars()

			got := strings.Join(l.envs, ",")
			want := test.want

			if got != want {
				t.Errorf("got %q want %q", got, want)
			}
		})
	}
}

func TestCommandProxy(t *testing.T) {

	var tests = []struct {
		description string
		enabled     bool
		port        string
		want        string
	}{
		{
			"No lope server env is added if it isn't enabled",
			false,
			"",
			"",
		},
		{
			"When lope server is enabled env var is exported",
			true,
			"8000",
			"8000",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			l.cfg.cmdProxy = test.enabled
			l.cfg.cmdProxyPort = test.port
			l.commandProxy()

			got := ""
			want := test.want
			for _, e := range l.params {
				split := strings.Split(e, "=")
				if split[0] == "LOPE_PROXY_ADDR" {
					addr, err := url.Parse(split[1])
					if err != nil {
						t.Errorf("got %q want %q", err, want)
					}
					got = addr.Port()

				}
			}
			if got != want {
				t.Errorf("got %q want %q", got, want)
			}
		})
	}
}
