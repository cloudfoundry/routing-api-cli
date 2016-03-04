package commands

import (
	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/db"
)

func Register(client routing_api.Client, routes []db.Route) error {
	return client.UpsertRoutes(routes)
}
