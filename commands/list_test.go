package commands_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/routing-api-cli/commands"
	"github.com/cloudfoundry-incubator/routing-api/fake_routing_api"
	"github.com/cloudfoundry-incubator/routing-api/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe(".List", func() {
	var (
		client *fake_routing_api.FakeClient
		route  models.Route
		routes []models.Route
	)

	BeforeEach(func() {
		client = &fake_routing_api.FakeClient{}
		route = models.Route{
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
		routesList, _ := commands.List(client)
		Expect(client.RoutesCallCount()).To(Equal(1))
		Expect(routesList).To(Equal(routes))
	})

})
