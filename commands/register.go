package commands

import (
	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/db"
	token_fetcher "github.com/cloudfoundry-incubator/uaa-token-fetcher"
)

func Register(client routing_api.Client, tokenFetcher token_fetcher.TokenFetcher, routes []db.Route) error {
	token, err := tokenFetcher.FetchToken(false)
	if err != nil {
		return err
	}
	client.SetToken(token.AccessToken)
	return client.UpsertRoutes(routes)
}
