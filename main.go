package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api-cli/commands"
	"github.com/cloudfoundry-incubator/routing-api/db"
	trace "github.com/cloudfoundry-incubator/trace-logger"
	token_fetcher "github.com/cloudfoundry-incubator/uaa-token-fetcher"
	"github.com/codegangsta/cli"
)

const RTR_TRACE = "RTR_TRACE"

var flags = []cli.Flag{
	cli.StringFlag{
		Name:  "api",
		Usage: "Endpoint for the routing-api. (required)",
	},
	cli.StringFlag{
		Name:  "client-id",
		Usage: "Id of the OAuth client. (required)",
	},
	cli.StringFlag{
		Name:  "client-secret",
		Usage: "Secret for OAuth client. (required)",
	},
	cli.StringFlag{
		Name:  "oauth-url",
		Usage: "URL for OAuth client. (required)",
	},
}

var eventsFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "http",
		Usage: "Stream HTTP events",
	},
	cli.BoolFlag{
		Name:  "tcp",
		Usage: "Stream TCP events",
	},
}

var cliCommands = []cli.Command{
	{
		Name:  "register",
		Usage: "Registers routes with the routing-api",
		Description: `Routes must be specified in JSON format, like so:
'[{"route":"foo.com", "port":12345, "ip":"1.2.3.4", "ttl":5, "log_guid":"log-guid"}]'`,
		Action: registerRoutes,
		Flags:  flags,
	},
	{
		Name:  "unregister",
		Usage: "Unregisters routes with the routing-api",
		Description: `Routes must be specified in JSON format, like so:
'[{"route":"foo.com", "port":12345, "ip":"1.2.3.4"]'`,
		Action: unregisterRoutes,
		Flags:  flags,
	},
	{
		Name:   "list",
		Usage:  "Lists the currently registered routes",
		Action: listRoutes,
		Flags:  flags,
	},
	{
		Name:   "events",
		Usage:  "Stream events from the Routing API",
		Action: streamEvents,
		Flags:  append(flags, eventsFlags...),
	},
}

var environmentVariableHelp = `ENVIRONMENT VARIABLES:
   RTR_TRACE=true	Print API request diagnostics to stdout`

func main() {
	fmt.Println()
	app := cli.NewApp()
	app.Name = "rtr"
	app.Usage = "A CLI for the Router API server."
	authors := []cli.Author{cli.Author{Name: "Cloud Foundry Routing Team", Email: "cf-dev@lists.cloudfoundry.org"}}
	app.Authors = authors
	app.Commands = cliCommands
	app.CommandNotFound = commandNotFound
	app.Version = "2.2.1"

	cli.AppHelpTemplate = cli.AppHelpTemplate + environmentVariableHelp + "\n"

	trace.NewLogger(os.Getenv(RTR_TRACE))

	app.Run(os.Args)
	os.Exit(0)
}

func registerRoutes(c *cli.Context) {
	issues := checkFlags(c)
	issues = append(issues, checkArguments(c, "register")...)

	if len(issues) > 0 {
		printHelpForCommand(c, issues, "register")
	}

	client := routing_api.NewClient(c.String("api"))

	config := buildOauthConfig(c)
	fetcher := token_fetcher.NewTokenFetcher(&config)

	desiredRoutes := c.Args().First()
	var routes []db.Route

	err := json.Unmarshal([]byte(desiredRoutes), &routes)
	if err != nil {
		fmt.Println("Invalid json format.")
		os.Exit(3)
	}

	err = commands.Register(client, fetcher, routes)
	if err != nil {
		fmt.Println("route registration failed:", err)
		os.Exit(3)
	}

	fmt.Printf("Successfully registered routes: %s\n", desiredRoutes)
}

func unregisterRoutes(c *cli.Context) {
	issues := checkFlags(c)
	issues = append(issues, checkArguments(c, "unregister")...)

	if len(issues) > 0 {
		printHelpForCommand(c, issues, "unregister")
	}

	client := routing_api.NewClient(c.String("api"))

	config := buildOauthConfig(c)
	fetcher := token_fetcher.NewTokenFetcher(&config)

	desiredRoutes := c.Args().First()
	var routes []db.Route
	err := json.Unmarshal([]byte(desiredRoutes), &routes)
	if err != nil {
		fmt.Println("Invalid json format.")
		os.Exit(3)
	}

	err = commands.UnRegister(client, fetcher, routes)
	if err != nil {
		fmt.Println("route unregistration failed:", err)
		os.Exit(3)
	}

	fmt.Printf("Successfully unregistered routes: %s\n", desiredRoutes)
}

func listRoutes(c *cli.Context) {
	issues := checkFlags(c)
	issues = append(issues, checkArguments(c, "list")...)

	if len(issues) > 0 {
		printHelpForCommand(c, issues, "list")
	}

	client := routing_api.NewClient(c.String("api"))

	config := buildOauthConfig(c)
	fetcher := token_fetcher.NewTokenFetcher(&config)
	routes, err := commands.List(client, fetcher)
	if err != nil {
		fmt.Println("listing routes failed:", err)
		os.Exit(3)
	}

	prettyRoutes, _ := json.Marshal(routes)

	fmt.Printf("%v\n", string(prettyRoutes))
}

func streamEvents(c *cli.Context) {
	issues := checkFlags(c)
	issues = append(issues, checkArguments(c, "events")...)

	if len(issues) > 0 {
		printHelpForCommand(c, issues, "events")
	}

	streamHttp := c.Bool("http")
	streamTcp := c.Bool("tcp")

	if !streamHttp && !streamTcp {
		streamHttp = true
		streamTcp = true
	}

	client := routing_api.NewClient(c.String("api"))

	config := buildOauthConfig(c)
	fetcher := token_fetcher.NewTokenFetcher(&config)
	token, err := fetcher.FetchToken()
	if err != nil {
		fmt.Println("Error fetching oauth token:", err)
		os.Exit(3)
	}

	client.SetToken(token.AccessToken)
	errorChan := make(chan error)
	eventChan := make(chan string)

	numOfSubscriptions := 0

	if streamHttp {
		numOfSubscriptions++
		go streamHttpEvents(client, eventChan, errorChan)
	}

	if streamTcp {
		numOfSubscriptions++
		go streamTcpEvents(client, eventChan, errorChan)
	}

	errorCount := 0

loop:
	for {
		select {
		case eventMessage := <-eventChan:
			fmt.Println(eventMessage)
		case err := <-errorChan:
			errorCount++
			fmt.Printf("Connection closed: %s", err.Error())
			if errorCount >= numOfSubscriptions {
				break loop
			}
		}
	}
}

func streamHttpEvents(client routing_api.Client, eventChan chan string, errorChan chan error) {
	eventSource, err := client.SubscribeToEvents()
	if err != nil {
		fmt.Println("streaming events failed:", err)
		return
	}
	for {
		e, err := eventSource.Next()
		if err != nil {
			errorChan <- err
			break
		}

		event, _ := json.Marshal(e)
		eventChan <- fmt.Sprintf("%v\n", string(event))
	}
}

func streamTcpEvents(client routing_api.Client, eventChan chan string, errorChan chan error) {
	eventSource, err := client.SubscribeToTcpEvents()
	if err != nil {
		fmt.Println("streaming events failed:", err)
		return
	}
	for {
		e, err := eventSource.Next()
		if err != nil {
			errorChan <- err
			break
		}

		event, _ := json.Marshal(e)
		eventChan <- fmt.Sprintf("%v\n", string(event))
	}
}

func buildOauthConfig(c *cli.Context) token_fetcher.OAuthConfig {
	var port int
	oauthUrl, _ := url.Parse(c.String("oauth-url"))
	addr := strings.Split(oauthUrl.Host, ":")
	host := addr[0]

	if len(addr) > 1 {
		port, _ = strconv.Atoi(addr[1])
	} else {
		if strings.ToLower(oauthUrl.Scheme) == "https" {
			port = 443
		} else if strings.ToLower(oauthUrl.Scheme) == "http" {
			port = 80
		}
	}

	return token_fetcher.OAuthConfig{
		TokenEndpoint: oauthUrl.Scheme + "://" + host,
		ClientName:    c.String("client-id"),
		ClientSecret:  c.String("client-secret"),
		Port:          port,
	}
}

func checkFlags(c *cli.Context) []string {
	var issues []string

	if c.String("api") == "" {
		issues = append(issues, "Must provide an API endpoint for the routing-api component.")
	}

	if c.String("client-id") == "" {
		issues = append(issues, "Must provide the id of an OAuth client.")
	}

	if c.String("client-secret") == "" {
		issues = append(issues, "Must provide an OAuth secret.")
	}

	if c.String("oauth-url") == "" {
		issues = append(issues, "Must provide an URL to the OAuth client.")
	}

	_, err := url.Parse(c.String("oauth-url"))
	if err != nil {
		issues = append(issues, "Invalid OAuth client URL")
	}

	return issues
}

func checkArguments(c *cli.Context, cmd string) []string {
	var issues []string

	switch cmd {
	case "register", "unregister":
		if len(c.Args()) > 1 {
			issues = append(issues, "Unexpected arguments.")
		} else if len(c.Args()) < 1 {
			issues = append(issues, "Must provide routes JSON.")
		}
	case "list", "events":
		if len(c.Args()) > 0 {
			issues = append(issues, "Unexpected arguments.")
		}
	}

	return issues
}

func printHelpForCommand(c *cli.Context, issues []string, cmd string) {
	for _, issue := range issues {
		fmt.Println(issue)
	}
	fmt.Println()
	cli.ShowCommandHelp(c, cmd)
	os.Exit(1)
}

func commandNotFound(c *cli.Context, cmd string) {
	fmt.Println("Not a valid command:", cmd)
	os.Exit(1)
}
