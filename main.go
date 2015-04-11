package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api-cli/commands"
	"github.com/cloudfoundry-incubator/routing-api/db"
	token_fetcher "github.com/cloudfoundry-incubator/uaa-token-fetcher"
)

var (
	apiEndpoint   = flag.String("api", "", "The api endpoint")
	oauthName     = flag.String("oauth-name", "", "")
	oauthPassword = flag.String("oauth-password", "", "")
	oauthURL      = flag.String("oauth-url", "", "")
	oauthPort     = flag.Int("oauth-port", 0, "")
)

func main() {
	fmt.Println(os.Args)
	cmd := os.Args[1]
	validateCommand(cmd)

	err := flag.CommandLine.Parse(os.Args[2:])
	if err != nil {
		fmt.Println("Error parsing flags:", err)
		os.Exit(1)
	}
	issues := checkFlags()

	if flag.NArg() == 0 {
		issues = append(issues, "Must provide routes JSON.")
	}

	if len(issues) > 0 {
		for _, issue := range issues {
			fmt.Println(issue)
		}
		os.Exit(1)
	}

	runCommand(cmd)

	os.Exit(0)
}

func runCommand(cmd string) {
	client := routing_api.NewClient(*apiEndpoint)
	config := token_fetcher.OAuthConfig{
		TokenEndpoint: *oauthURL,
		ClientName:    *oauthName,
		ClientSecret:  *oauthPassword,
		Port:          *oauthPort,
	}

	fetcher := token_fetcher.NewTokenFetcher(&config)

	var routes []db.Route
	_ = json.Unmarshal([]byte(flag.Args()[0]), &routes)

	switch cmd {
	case "register":
		err := commands.Register(client, fetcher, routes)
		if err != nil {
			fmt.Println("route registration failed:", err)
			os.Exit(3)
		}
	case "unregister":
		err := commands.UnRegister(client, fetcher, routes)
		if err != nil {
			fmt.Println("route unregisterification failed:", err)
			os.Exit(3)
		}
	}
}

func validateCommand(cmd string) {
	switch cmd {
	case "register":
	case "unregister":
	default:
		fmt.Println("Not a valid command:", cmd)
	}
}

func checkFlags() []string {
	var issues []string
	if *apiEndpoint == "" {
		issues = append(issues, "Must provide an API endpoint for the routing-api component.\n")
	}

	if *oauthName == "" {
		issues = append(issues, "Must provide the name of an OAuth client.\n")
	}

	if *oauthPassword == "" {
		issues = append(issues, "Must provide an OAuth password/secret.\n")
	}

	if *oauthURL == "" {
		issues = append(issues, "Must provide an URL to the OAuth client.\n")
	}

	if *oauthPort == 0 {
		issues = append(issues, "Must provide the port the OAuth client is listening on.\n")
	}
	return issues
}
