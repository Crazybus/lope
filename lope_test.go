package main

import (
	"fmt"
	"strings"
	"testing"
)

var c = &config{
	cmd:          []string{"ls", "-lhatr"},
	entrypoint:   "/bin/sh",
	envBlacklist: []string{"HOME"},
	envPattern:   "VAULT|AWS|GOOGLE_|GITHUB",
	home:         "/home/lope",
	image:        "lopeImage",
	paths: []string{
		path(".vault-token"),
		path(".aws/"),
		path(".kube/"),
		path(".ssh/"),
	},
}

var l = lope{
	cfg: c,
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
		want        string
	}{
		{
			"Add the aws directory",
			[]string{".aws"},
			path("./test/"),
			fmt.Sprintf("-v %v.aws:/root/.aws", path("./test/")),
		},
		{
			"Don't add any directories if they don't exist",
			[]string{".aws", ".not-exist"},
			path("./test/"),
			fmt.Sprintf("-v %v.aws:/root/.aws", path("./test/")),
		},
		{
			"Add multiple directories",
			[]string{".aws", ".kube"},
			path("./test/"),
			fmt.Sprintf("-v %v.aws:/root/.aws -v %v.kube:/root/.kube", path("./test/"), path("./test/")),
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			l.params = make([]string, 0)
			l.cfg.home = test.home
			l.cfg.paths = test.paths
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
		pattern     string
		want        string
	}{
		{
			"Add an env var",
			[]string{"ENV1=hello1"},
			[]string{""},
			"",
			"-e ENV1=hello1",
		},
		{
			"Add an multiple env vars",
			[]string{"ENV1=hello1", "ENV2=hello2"},
			[]string{""},
			"",
			"-e ENV1=hello1 -e ENV2=hello2",
		},
		{
			"Blacklist an env var",
			[]string{"ENV1=hello1"},
			[]string{"ENV1"},
			"",
			"",
		},
		{
			"Whitelist an env var",
			[]string{"ENV1=hello1", "ENV2=hello2", "NO=no"},
			[]string{""},
			"ENV",
			"-e ENV1=hello1 -e ENV2=hello2",
		},
		{
			"Blacklist and whitelisting env vars",
			[]string{"ENV1=hello1", "ENV2=hello2", "NO=no"},
			[]string{"ENV1"},
			"ENV",
			"-e ENV2=hello2",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			l.params = make([]string, 0)
			l.envs = test.envs
			l.cfg.envBlacklist = test.blacklist
			l.cfg.envPattern = test.pattern
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
			"Override the entrypoing",
			"/bin/ohyeah",
			"run --rm --entrypoint /bin/ohyeah -w /lope",
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