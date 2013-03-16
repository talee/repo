// Creates repositories remotely via command-line options. Uses the OS's
// keychain for authentication.
//
// Usage:
// 	repo [OPTIONS] [-d DESCRIPTION]
//
// Example:
// 	// Create a private C++ repository called gorepo.
// 
// 	repo create -n gorepo -l c++ -p -d Best repo ever
//
// 	// Create a public Scala repository with git for source control.
//
// 	repo -c -n gorepo -l scala -s git
//
package main

import (
	"bitbucket.org/tlee/netgo/inspect"
	"bitbucket.org/tlee/netgo/keychain"
	"fmt"
	"github.com/gaal/go-options/options"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	BITBUCKET_ENDPOINT = "https://api.bitbucket.org/1.0/"
	BITBUCKET_REPO     = "repositories"
	BITBUCKET_HOST     = "bitbucket.org"
)

var spec = options.NewOptions(`
usage: repo [OPTIONS] [-d DESCRIPTION]
repo - Create repositories remotely.

	example:
	Create a private c++ repository called gorepo.

	repo -c -n gorepo -l c++ -p -d Best repo ever

--
c,create        Create a repository
n,name=         Name of the repository
d,description=  Description of the repository. Should be the last flag
s,scm=          Source control type [hg]
l,lang,language=     Coding language (must be lowercase) [go]
p,is_private    True if repository is hidden from public. Default false
h,help          Show usage
`)

func main() {
	// Get command based on command-line flags
	cmd := getCommand()
	err := cmd.Exec()
	handle(err, "Failed execution.")
	fmt.Println("Created new repository.\n")
	printFormValues(cmd.FormValues)

	fmt.Println("\nDone")
}

// Pretty print for url.Values
func printFormValues(pairs url.Values) {
	var keys []string
	for key, _ := range pairs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Printf("%-16s %s\n", key+":", pairs[key][0])
	}
}

// Command to send an authenticated form with flags from the command-line.
type NetCommand struct {
	// Allows NetCommand to parse command-line flags
	*options.OptionSpec
	Name               string
	HttpMethod         string
	Url                string // RESTful endpoint
	CredentialHostname string // Hostname of the form receiver e.g. "bitbucket.org"
	FormValues         url.Values
	Flags              options.Options
}

// Executes the command and sends the authenticated form. Redirects up to 5 times as neccessary.
func (c *NetCommand) Exec() (err error) {
	resp, err := c.do()
	handle(err, "Failed to send response")
	if resp.StatusCode >= http.StatusBadRequest {
		fmt.Fprintf(os.Stderr, "Failed request. Got a bad status code\n\n.")
		printResponse(resp)
		inspect.Header(resp.Request.Header, os.Stderr)
		os.Exit(3)
	}
	var respErr error
	for i := 0; i < 5 && respErr == nil && (resp.StatusCode >= http.StatusMovedPermanently && resp.StatusCode <= http.StatusTemporaryRedirect || resp.StatusCode == http.StatusUnauthorized); i++ {
		req := resp.Request
		if resp.StatusCode != http.StatusUnauthorized {
			req.URL, err = resp.Location()
			if err != nil {
				fmt.Fprintln(os.Stderr, "No redirect location found!\n")
				os.Exit(3)
			}
		}
		req.Host = req.URL.Host
		fmt.Printf("Redirecting to %s\n\n", req.URL)
		fmt.Printf("Request header: %v\n\n", req.Header)
		resp, respErr = http.DefaultClient.Do(req)
		printResponse(resp)
	}
	defer resp.Body.Close()
	return err
}

func printResponse(resp *http.Response) {
	respDump, _ := httputil.DumpResponse(resp, true)
	fmt.Printf("%s\n", respDump)
	req := resp.Request
	reqDump, _ := httputil.DumpRequest(req, true)
	fmt.Printf("%s\n", reqDump)
	b, err := ioutil.ReadAll(req.Body)
	handle(err, "Can't read request body")
	fmt.Printf("BODY (%v): %s\n", len(b), b)
}

func (c *NetCommand) do() (*http.Response, error) {
	req, err := http.NewRequest(c.HttpMethod, c.Url, strings.NewReader(c.FormValues.Encode()))
	handle(err, "Failed to create request for "+c.Name)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	// Get authentication
	username, pw, err := keychain.Credentials(c.CredentialHostname)
	handle(err, "Failed to get keychain credentials.")
	req.SetBasicAuth(username, pw)
	// Fire request
	return http.DefaultClient.Do(req)
}

func getCommand() *NetCommand {
	// User must specify a command
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "create", "--create", "-c", "c":
			return getCreateCmd()
		default:
			printUsage()
		}
	}
	printUsage()
	panic("invalid getCommand")
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: repo create\n\n\"repo [command]\" for details") // Change to template later
	os.Exit(2)
}

// Contains the form keys needed to create a repository.
type RepositoryForm struct {
	name        string
	description string
	scm         string
	language    string
	is_private  bool
}

func getCreateCmd() *NetCommand {
	// Define flags
	createCmd := &NetCommand{
		Name:               "create",
		HttpMethod:         "POST",
		CredentialHostname: BITBUCKET_HOST,
		Url:                BITBUCKET_ENDPOINT + BITBUCKET_REPO,
		OptionSpec:         spec,
	}
	// Get flags
	createCmd.Flags = createCmd.OptionSpec.Parse(os.Args[2:])
	repo := RepositoryForm{
		name:        createCmd.Flags.Get("name"),
		description: strings.Join(append([]string{createCmd.Flags.Get("description")}, createCmd.Flags.Extra...), " "),
		scm:         createCmd.Flags.Get("scm"),
		language:    createCmd.Flags.Get("language"),
		is_private:  createCmd.Flags.GetBool("is_private"),
	}
	// Require repo name
	if len(repo.name) == 0 {
		if createCmd.Flags.GetBool("help") {
			createCmd.OptionSpec.PrintUsageAndExit("")
		} else {
			createCmd.OptionSpec.PrintUsageAndExit("A repository name is required [-n name]")
		}
	}
	// Create form values
	createCmd.FormValues = url.Values{
		"name":        {repo.name},
		"description": {repo.description},
		"scm":         {repo.scm},
		"language":    {repo.language},
		"is_private":  {strconv.FormatBool(repo.is_private)},
	}
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
