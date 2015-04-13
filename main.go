package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api-cli/commands"
	"github.com/cloudfoundry-incubator/routing-api/db"
	token_fetcher "github.com/cloudfoundry-incubator/uaa-token-fetcher"
	"github.com/codegangsta/cli"
)

var flags = []cli.Flag{
	cli.StringFlag{
		Name:  "api",
		Usage: "Endpoint for the routing-api.",
	},
	cli.StringFlag{
		Name:  "oauth-name",
		Usage: "Name of the OAuth client.",
	},
	cli.StringFlag{
		Name:  "oauth-password",
		Usage: "Password for OAuth client.",
	},
	cli.StringFlag{
		Name:  "oauth-url",
		Usage: "URL for OAuth client.",
	},
	cli.IntFlag{
		Name:  "oauth-port",
		Usage: "Port OAuth client is listening on.",
	},
}

var cliCommands = []cli.Command{
	{
		Name:   "register",
		Usage:  "Registers routes with the routing-api",
		Action: registerRoutes,
		Flags:  flags,
	},
	{
		Name:   "unregister",
		Usage:  "Unregisters routes with the routing-api",
		Action: unregisterRoutes,
		Flags:  flags,
	},
}

func main() {
	app := cli.NewApp()
	app.Name = "rtr"
	app.Commands = cliCommands
	app.CommandNotFound = commandNotFound

	app.Run(os.Args)
	os.Exit(0)
}

func registerRoutes(c *cli.Context) {
	checkFlagsAndArguments(c)

	client := routing_api.NewClient(c.String("api"))

	config := buildOauthConfig(c)
	fetcher := token_fetcher.NewTokenFetcher(&config)

	var routes []db.Route
	_ = json.Unmarshal([]byte(c.Args().First()), &routes)

	err := commands.Register(client, fetcher, routes)
	if err != nil {
		fmt.Println("route registration failed:", err)
		os.Exit(3)
	}
}

func unregisterRoutes(c *cli.Context) {
	checkFlagsAndArguments(c)

	client := routing_api.NewClient(c.String("api"))

	config := buildOauthConfig(c)
	fetcher := token_fetcher.NewTokenFetcher(&config)

	var routes []db.Route
	_ = json.Unmarshal([]byte(c.Args().First()), &routes)

	err := commands.UnRegister(client, fetcher, routes)
	if err != nil {
		fmt.Println("route unregistration failed:", err)
		os.Exit(3)
	}
}

func buildOauthConfig(c *cli.Context) token_fetcher.OAuthConfig {
	return token_fetcher.OAuthConfig{
		TokenEndpoint: c.String("oauth-url"),
		ClientName:    c.String("oauth-name"),
		ClientSecret:  c.String("oauth-password"),
		Port:          c.Int("oauth-port"),
	}
}

func checkFlagsAndArguments(c *cli.Context) {
	var issues []string

	if c.String("api") == "" {
		issues = append(issues, "Must provide an API endpoint for the routing-api component.\n")
	}

	if c.String("oauth-name") == "" {
		issues = append(issues, "Must provide the name of an OAuth client.\n")
	}

	if c.String("oauth-password") == "" {
		issues = append(issues, "Must provide an OAuth password/secret.\n")
	}

	if c.String("oauth-url") == "" {
		issues = append(issues, "Must provide an URL to the OAuth client.\n")
	}

	if c.Int("oauth-port") == 0 {
		issues = append(issues, "Must provide the port the OAuth client is listening on.\n")
	}

	if !c.Args().Present() {
		issues = append(issues, "Must provide routes JSON. \n")
	}

	if len(issues) > 0 {
		for _, issue := range issues {
			fmt.Println(issue)
		}
		os.Exit(1)
	}
}

func commandNotFound(c *cli.Context, cmd string) {
	fmt.Println("Not a valid command:", cmd)
	os.Exit(1)
}
