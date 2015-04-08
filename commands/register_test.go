package commands_test

import (
	"github.com/cloudfoundry-incubator/routing-api-cli/commands"
	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/cloudfoundry-incubator/routing-api/fake_routing_api"
	"github.com/cloudfoundry/gorouter/token_fetcher"
	fake_token_fetcher "github.com/cloudfoundry/gorouter/token_fetcher/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe(".Register", func() {
	var (
		client       *fake_routing_api.FakeClient
		tokenFetcher *fake_token_fetcher.FakeTokenFetcher
	)

	BeforeEach(func() {
		client = &fake_routing_api.FakeClient{}
		tokenFetcher = &fake_token_fetcher.FakeTokenFetcher{}
		tokenFetcher.FetchTokenReturns(&token_fetcher.Token{AccessToken: "token"}, nil)
	})

	It("registers routes", func() {
		routes := []db.Route{{}}
		commands.Register(client, tokenFetcher, routes)
		Expect(client.UpsertRoutesCallCount()).To(Equal(1))
		Expect(client.UpsertRoutesArgsForCall(0)).To(Equal(routes))
	})
})
