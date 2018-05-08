package main

import (
	"fmt"
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
			"-v /home/user/pro/lope/:/lope/",
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
			"-e ENV1=hello1",
		},
		{
			"Add multiple env vars",
			[]string{"ENV1=hello1", "ENV2=hello2"},
			[]string{},
			[]string{},
			false,
			"-e ENV1=hello1 -e ENV2=hello2",
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
			"-e ENV1=hello1 -e ENV2=hello2",
		},
		{
			"Blacklist and whitelisting env vars",
			[]string{"ENV1=hello1", "ENV2=hello2", "NO=no"},
			[]string{"ENV1"},
			[]string{"ENV"},
			false,
			"-e ENV2=hello2",
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
		want        string
	}{
		{
			"Override the entrypoint",
			"/bin/ohyeah",
			"docker run --rm --entrypoint /bin/ohyeah -w /lope",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			l.params = make([]string, 0)
			l.cfg.entrypoint = test.entrypoint
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
		instructions []string
		want         []string
	}{
		{
			"Create a basic dockerfile",
			"imageName",
			false,
			true,
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
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			l.cfg.sourceImage = test.image
			l.cfg.instructions = test.instructions
			l.cfg.mount = test.mount
			l.cfg.addMount = test.addMount
			l.createDockerfile()

			got := l.dockerfile
			want := strings.Join(test.want, "\n")

			if got != want {
				t.Errorf("got %q want %q", got, want)
			}
		})
	}
}
