package commands

import (
	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/db"
	token_fetcher "github.com/cloudfoundry-incubator/uaa-token-fetcher"
)

func List(client routing_api.Client, tokenFetcher token_fetcher.TokenFetcher) ([]db.Route, error) {
	token, err := tokenFetcher.FetchToken()
	if err != nil {
		return nil, err
	}
	client.SetToken(token.AccessToken)
	return client.Routes()
}
