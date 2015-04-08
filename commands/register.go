package commands

import (
	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/cloudfoundry/gorouter/token_fetcher"
)

func Register(client routing_api.Client, tokenFetcher token_fetcher.TokenFetcher, routes []db.Route) error {
	token, err := tokenFetcher.FetchToken()
	if err != nil {
		return err
	}
	client.SetToken(token.AccessToken)
	return client.UpsertRoutes(routes)
}
