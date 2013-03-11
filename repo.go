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
	cmd := getCommand()
	fmt.Printf("%s\n", cmd.JsonArgs)
}

type NetCommand struct {
	*flag.FlagSet
	Name       string
	HttpMethod string
	Url        string
	JsonArgs   []byte
}

func (c *NetCommand) Exec() (err error) {
	req, err := http.NewRequest(c.HttpMethod, c.Url, nil)
	handle(err, "Failed to create request for "+c.Name)
	resp, err := http.DefaultClient.Do(req)
	respDump, _ := httputil.DumpResponse(resp, true)
	fmt.Printf("%s", respDump)
	return err
}

/*
func NewNetCommand() *NetCommand {
	c := &NetCommand{}
	c.FlagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		c.FlagSet.PrintDefaults()
	}
	return c
}
*/

type Repository struct {
	Name,
	Description,
	Scm,
	Language string
	IsPrivate bool
}

func getCommand() *NetCommand {
	// User must specify a command
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "create":
			return getCreateCmd()
		default:
			printUsage()
		}
	}
	printUsage()
	panic("invalid getCommand")
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: repo create\n\n\"repo [command] -help\" for details") // Change to template later
	os.Exit(2)
}

func getCreateCmd() *NetCommand {
	// Define flags
	repo := Repository{}
	createCmd := &NetCommand{Name: "create", flag.FlagSet: flag.NewFlagSet("create", flag.ExitOnError)}
	createCmd.StringVar(&repo.Name, "n", "", "name of the repository. Required")
	createCmd.StringVar(&repo.Description, "d", "", "description of the repository")
	createCmd.StringVar(&repo.Scm, "scm", "hg", "git|hg. Default hg")
	createCmd.StringVar(&repo.Language, "lang", "", "coding language")
	createCmd.BoolVar(&repo.IsPrivate, "p", false, "is repository private. Default false (publicly viewable)")
	// Get flags
	createCmd.Parse(os.Args[2:])

	// Require repo name
	if len(repo.Name) == 0 {
		fmt.Fprintln(os.Stderr, "A repository name is required [-n name]\n")
		createCmd.FlagSet.PrintDefaults()
		os.Exit(2)
	}
	// Marshall arguments into JSON and store in command
	jsonArgs, err := json.Marshal(repo)
	handle(err, "Failed to marshall arguments")
	createCmd.JsonArgs = jsonArgs
	return createCmd
}

func handle(err error, msg string) {
	if err != nil {
		if len(msg) > 0 {
			fmt.Fprintf(os.Stderr, "\nERROR: %v\n\n", msg)
		}
		fmt.Fprintf(os.Stderr, "%v\n\n", err)
		os.Exit(2)
	}
}
