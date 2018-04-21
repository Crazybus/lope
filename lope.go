package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

// https://npf.io/2015/06/testing-exec-command/

func dockerRun(args []string) string {
	cmd := exec.Command("docker", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(out.String())
	}
	return out.String()
}

func buildImage(image string) string {
	lopeImage := "lope"
	file, err := ioutil.TempFile(os.TempDir(), lopeImage)
	defer os.Remove(file.Name())

	_, err = file.WriteString(fmt.Sprintf("FROM %v\n ADD . /lope", image))
	if err != nil {
		log.Fatal(err)
	}

	build := make([]string, 0)
	build = append(build, "build", "-t", lopeImage, "-f", file.Name(), ".")
	fmt.Println(dockerRun(build))
	return lopeImage
}

func addEnvVars(run []string) []string {
	blacklist := []string{"HOME"}
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		blacklisted := false
		for _, b := range blacklist {
			if pair[0] == b {
				blacklisted = true
				break
			}
		}
		if !blacklisted {
			run = append(run, "-e", e)
		} else {
			fmt.Println("Not adding environment variable:", e)
		}
	}
	return run
}

func main() {

	image := buildImage(os.Args[1])

	run := make([]string, 0)
	run = append(run, "run", "--rm", "-w", "/lope")
	run = addEnvVars(run)
	run = append(run, "-v", "/Users/mick/.vault-token:/root/.vault-token")
	run = append(run, image, "sh", "-c", strings.Join(os.Args[2:], " "))
	fmt.Println(dockerRun(run))
}
