package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api-cli/commands"
	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/cloudfoundry/gorouter/config"
	"github.com/cloudfoundry/gorouter/token_fetcher"
)

var (
	apiEndpoint   = flag.String("api", "", "The api endpoint")
	oauthName     = flag.String("oauth-name", "", "")
	oauthPassword = flag.String("oauth-password", "", "")
	oauthURL      = flag.String("oauth-url", "", "")
	oauthPort     = flag.Int("oauth-port", 0, "")
)

type OAuthConfig struct {
	TokenEndpoint string `yaml:"token_endpoint"`
	ClientName    string `yaml:"client_name"`
	ClientSecret  string `yaml:"client_secret"`
	Port          int    `yaml:"port"`
}

func main() {
	flag.Parse()

	issues := checkFlags()

	if flag.NArg() == 0 {
		issues = append(issues, "please provide routes body")
	}

	if len(issues) > 0 {
		for _, issue := range issues {
			fmt.Println(issue)
		}
		os.Exit(1)
	}

	client := routing_api.NewClient(*apiEndpoint)
	config := config.OAuthConfig{
		TokenEndpoint: *oauthURL,
		ClientName:    *oauthName,
		ClientSecret:  *oauthPassword,
		Port:          *oauthPort,
	}
	fetcher := token_fetcher.NewTokenFetcher(&config)

	var routes []db.Route
	_ = json.Unmarshal([]byte(flag.Args()[0]), &routes)
	err := commands.Register(client, fetcher, routes)
	if err != nil {
		fmt.Println("route registration failed:", err)
		os.Exit(3)
	}

	os.Exit(0)
}

func checkFlags() []string {
	var issues []string
	if *apiEndpoint == "" {
		issues = append(issues, "please provide an api endpoint")
	}

	if *oauthName == "" {
		issues = append(issues, "please provide an oauth-name")
	}

	if *oauthPassword == "" {
		issues = append(issues, "please provide an oauth-password")
	}

	if *oauthURL == "" {
		issues = append(issues, "please provide an oauth-url")
	}

	if *oauthPort == 0 {
		issues = append(issues, "please provide an oauth-port")
	}
	return issues
}
