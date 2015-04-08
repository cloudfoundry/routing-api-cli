package main

import (
	"encoding/json"
	"os"
	"strconv"

	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api-cli/commands"
	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/cloudfoundry/gorouter/config"
	"github.com/cloudfoundry/gorouter/token_fetcher"
)

type OAuthConfig struct {
	TokenEndpoint string `yaml:"token_endpoint"`
	ClientName    string `yaml:"client_name"`
	ClientSecret  string `yaml:"client_secret"`
	Port          int    `yaml:"port"`
}

func main() {
	client := routing_api.NewClient(os.Args[2])
	port, _ := strconv.Atoi(os.Args[10])
	config := config.OAuthConfig{
		TokenEndpoint: os.Args[8],
		ClientName:    os.Args[4],
		ClientSecret:  os.Args[6],
		Port:          port,
	}
	fetcher := token_fetcher.NewTokenFetcher(&config)

	var routes []db.Route
	_ = json.Unmarshal([]byte(os.Args[len(os.Args)-1]), &routes)
	commands.Register(client, fetcher, routes)

	os.Exit(0)
}
