package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryName = filepath.FromSlash("./lope")

func TestLopeCli(t *testing.T) {

	var tests = []struct {
		description string
		cmd         []string
		match       []string
	}{
		{
			"Run lope with no arguments and get the help message",
			[]string{},
			[]string{
				"Usage of lope",
			},
		},
		{
			"Run a basic alpine image",
			[]string{
				"-noTty",
				"alpine",
				"ls",
			},
			[]string{
				"README.md",
			},
		},
		{
			"Run docker inside lope",
			[]string{
				"-noTty",
				"-addDocker",
				"alpine",
				"docker",
				"ps",
			},
			[]string{
				"CONTAINER",
				"STATUS",
				"lope",
			},
		},
		{
			"Run a command via the command proxy",
			[]string{
				"-noTty",
				"-cmdProxy",
				"alpine",
				"wget", "-q", "-O-",
				`--post-data='{"command":"lope", "args": ["-noTty", "alpine", "ls"]}'`,
				"--header=Content-Type:application/json",
				"http://$LOPE_PROXY_ADDR",
			},
			[]string{
				"README.md",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			cmd := exec.Command(binaryName, test.cmd...)
			out, _ := cmd.CombinedOutput()
			output := string(out)

			for _, match := range test.match {
				if !strings.Contains(output, match) {
					t.Errorf("expected %q to contain %q", output, match)
				}
			}
		})
	}
}
