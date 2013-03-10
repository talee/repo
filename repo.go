package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
)

const (
	BITBUCKET = "https://api.bitbucket.org/1.0/"
)

func main() {
	// Get flags set for command
	cmd, err := getCommand()
	handle(err, "Failed to get arguments as JSON")
	fmt.Printf("%v\n", cmd)
}

type NetCommand struct {
	name     string
	flagSet  flag.FlagSet
	method   string
	url      string
	jsonArgs []byte
}

func (c *NetCommand) Exec() (err error) {
	req, err := http.NewRequest(c.method, c.url, nil)
	handle(err, "Failed to create request for "+c.name)
	resp, err := http.DefaultClient.Do(req)
	respDump, _ := httputil.DumpResponse(resp, true)
	fmt.Printf("%s", respDump)
	return err
}

type Repository struct {
	name,
	description,
	scm,
	language string
	isPrivate bool
}

func getCommand() (*NetCommand, error) {
	// User must specify a command
	if len(os.Args) > 2 {
		switch os.Args[1] {
		case "create":
			return getCreateCmd(), nil
		}
	}
	return nil, fmt.Errorf("usage: repo create [--n name] [--d description] [--scm git|hg] [--lang code] [-p is_private]")
}

func getCreateCmd() *NetCommand {
	createFlags := flag.NewFlagSet("create", flag.ExitOnError)
	repo := Repository{
		name:        *createFlags.String("--n", "", "name of the repository"),
		description: *createFlags.String("--d", "", "description of the repository"),
		scm:         *createFlags.String("--scm", "hg", "git|hg. Default hg"),
	}
	jsonArgs, err := json.Marshal(repo)
	handle(err, "Failed to marshall arguments")
	createCmd := NetCommand{
		name:     "create",
		jsonArgs: jsonArgs,
	}

	return &createCmd
}

func handle(err error, msg string) {
	if err != nil {
		if len(msg) > 0 {
			fmt.Fprintf(os.Stderr, "\nERROR: %v\n\n", msg)
		}
		fmt.Fprintf(os.Stderr, "%v\n\n", err)
		os.Exit(1)
	}
}
