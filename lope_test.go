package main

import (
	"strings"
	"testing"
)

var config = &Config{
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

var lope = Lope{
	cfg: config,
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
			lope.params = make([]string, 0)
			lope.cfg.cmd = test.cmd
			lope.cfg.image = test.image
			lope.runParams()

			got := strings.Join(lope.params, " ")
			want := test.want

			if got != want {
				t.Errorf("got %q want %q", got, want)
			}
		})
	}
}
