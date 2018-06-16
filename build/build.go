package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

var buildDir = filepath.FromSlash("build/")

var operatingSystems = [...]string{
	"darwin",
	"linux",
	"windows",
}

var archs = [...]string{
	"386",
	"amd64",
}

func checksum(goos string, goarch string) error {
	file := buildDir + "lope-" + goos + "_" + goarch
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	sumFile := file + ".sha256"
	hash := fmt.Sprintf("%x", h.Sum(nil))
	fmt.Println(hash, file)

	hashFile, err := os.Create(sumFile)
	if err != nil {
		return err
	}
	defer hashFile.Close()

	_, err = hashFile.WriteString(hash)
	if err != nil {
		return err
	}
	return nil
}

func build(goos string, goarch string) error {
	os.Setenv("GOOS", goos)
	os.Setenv("GOARCH", goarch)

	cmd := exec.Command(
		"/usr/local/go/bin/go",
		"build",
		"-v",
		"-o",
		buildDir+"lope-"+goos+"_"+goarch,
	)
	return cmd.Run()
}

func main() {
	for _, goos := range operatingSystems {
		for _, goarch := range archs {
			err := build(goos, goarch)
			if err != nil {
				log.Printf("Failed to build %s/%s with error: %v", goos, goarch, err)
				os.Exit(1)
			}
			err = checksum(goos, goarch)
			if err != nil {
				log.Printf("Failed to generate checksums for %s/%s with error: %v", goos, goarch, err)
				os.Exit(1)
			}
		}
	}
}
