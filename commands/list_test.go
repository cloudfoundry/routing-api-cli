package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/routing-api-cli/commands"
	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/cloudfoundry-incubator/routing-api/fake_routing_api"
	token_fetcher "github.com/cloudfoundry-incubator/uaa-token-fetcher"
	fake_token_fetcher "github.com/cloudfoundry-incubator/uaa-token-fetcher/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe(".List", func() {
	var (
		client       *fake_routing_api.FakeClient
		tokenFetcher *fake_token_fetcher.FakeTokenFetcher
		route        db.Route
		routes       []db.Route
	)

	BeforeEach(func() {
		client = &fake_routing_api.FakeClient{}
		tokenFetcher = &fake_token_fetcher.FakeTokenFetcher{}
		tokenFetcher.FetchTokenReturns(&token_fetcher.Token{AccessToken: "token"}, nil)
		route = db.Route{
			Route:   "post_here",
			Port:    7000,
			IP:      "1.2.3.4",
			TTL:     50,
			LogGuid: "my-guid",
		}
		routes = append(routes, route)
		error := errors.New("this is an error")
		client.RoutesReturns(routes, error)
	})

	It("lists routes", func() {
		routesList, _ := commands.List(client, tokenFetcher)
		Expect(client.RoutesCallCount()).To(Equal(1))
		Expect(routesList).To(Equal(routes))
	})

	It("fetches a token and sets it on the client", func() {
		commands.List(client, tokenFetcher)
		Expect(client.SetTokenCallCount()).To(Equal(1))
		Expect(client.SetTokenArgsForCall(0)).To(Equal("token"))
	})
})
