package commands_test

import (
	"github.com/cloudfoundry-incubator/routing-api-cli/commands"
	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/cloudfoundry-incubator/routing-api/fake_routing_api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe(".UnRegister", func() {
	var (
		client *fake_routing_api.FakeClient
	)

	BeforeEach(func() {
		client = &fake_routing_api.FakeClient{}
	})

	It("unregisters routes", func() {
		routes := []db.Route{{}}
		commands.UnRegister(client, routes)
		Expect(client.DeleteRoutesCallCount()).To(Equal(1))
		Expect(client.DeleteRoutesArgsForCall(0)).To(Equal(routes))
	})

})
