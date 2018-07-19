package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

type lopeCmd struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

func run(cmd string, args []string, url string) {

	lopeCmd := lopeCmd{
		Command: cmd,
		Args:    args,
	}

	b, err := json.Marshal(lopeCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Print(string(body))
}

func main() {
	cmd := filepath.Base(os.Args[0])
	args := os.Args[1:]

	url, ok := os.LookupEnv("LOPE_PROXY_ADDR")
	if !ok {
		fmt.Println("Please set the 'LOPE_PROXY_ADDR' environment variable")
		os.Exit(1)
	}
	run(cmd, args, url)
}
