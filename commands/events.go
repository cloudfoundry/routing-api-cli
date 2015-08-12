package commands

import (
	"github.com/cloudfoundry-incubator/routing-api"
	token_fetcher "github.com/cloudfoundry-incubator/uaa-token-fetcher"
)

func Events(client routing_api.Client, tokenFetcher token_fetcher.TokenFetcher) (routing_api.EventSource, error) {
	token, err := tokenFetcher.FetchToken()
	if err != nil {
		return nil, err
	}
	client.SetToken(token.AccessToken)
	return client.SubscribeToEvents()
}
