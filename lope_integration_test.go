package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryName = filepath.FromSlash("./lope")

func init() {
	cmd := exec.Command("go", "build")
	err := cmd.Run()
	if err != nil {
		panic(err)
	}
	cmd = exec.Command("go", "build", "-o", "lope", "cmdProxy.go")
	cmd.Env = os.Environ()
	cmd.Env = append(
		cmd.Env,
		"GOOS=linux",
		"GOARCH=amd64",
		"CGO_ENABLED=0",
	)

	cmd.Dir = "cmdProxy"
	err = cmd.Run()
	if err != nil {
		panic(err)
	}
	if err := os.Chmod(filepath.FromSlash("cmdProxy/lope"), 0755); err != nil {
		panic(err)
	}
}

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
			"Run a command via the command proxy api",
			[]string{
				"-noTty",
				"-cmdProxy",
				"alpine",
				"wget", "-q", "-O-",
				`--post-data='{"command":"lope", "args": ["-noTty", "alpine", "ls"]}'`,
				"--header=Content-Type:application/json",
				"$LOPE_PROXY_ADDR",
			},
			[]string{
				"README.md",
			},
		},
		{
			"Run a command via the command proxy cli",
			[]string{
				"-noTty",
				"-cmdProxy",
				"alpine",
				"cmdProxy/lope", "-noTty", "alpine", "ls",
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
