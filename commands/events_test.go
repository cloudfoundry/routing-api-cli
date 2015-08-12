package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api-cli/commands"
	"github.com/cloudfoundry-incubator/routing-api/fake_routing_api"
	token_fetcher "github.com/cloudfoundry-incubator/uaa-token-fetcher"
	fake_token_fetcher "github.com/cloudfoundry-incubator/uaa-token-fetcher/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe(".Events", func() {
	var (
		client       *fake_routing_api.FakeClient
		tokenFetcher *fake_token_fetcher.FakeTokenFetcher
		eventSource  routing_api.EventSource
	)

	BeforeEach(func() {
		client = &fake_routing_api.FakeClient{}
		tokenFetcher = &fake_token_fetcher.FakeTokenFetcher{}
		tokenFetcher.FetchTokenReturns(&token_fetcher.Token{AccessToken: "token"}, nil)
		eventSource = &fake_routing_api.FakeEventSource{}
		client.SubscribeToEventsReturns(eventSource, nil)
	})

	It("list events", func() {
		eventsList, _ := commands.Events(client, tokenFetcher)
		Expect(client.SubscribeToEventsCallCount()).To(Equal(1))
		Expect(eventsList).To(Equal(eventSource))
	})

	It("fetches a token and sets it on the client", func() {
		commands.Events(client, tokenFetcher)
		Expect(client.SetTokenCallCount()).To(Equal(1))
		Expect(client.SetTokenArgsForCall(0)).To(Equal("token"))
	})

	Context("when SubscribeToEvents returns an error", func() {
		BeforeEach(func() {
			client.SubscribeToEventsReturns(nil, errors.New("Boom"))
		})

		It("returns an error as well", func() {
			_, err := commands.Events(client, tokenFetcher)
			Expect(client.SubscribeToEventsCallCount()).To(Equal(1))
			Expect(err).To(HaveOccurred())
		})
	})
})
