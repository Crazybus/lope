package main

import (
	"os/exec"
	"strings"
	"testing"
)

var binaryName = "lope"

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
				"-noSSH",
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
				"-noSSH",
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
