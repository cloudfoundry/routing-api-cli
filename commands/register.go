package commands

import (
	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/models"
)

func Register(client routing_api.Client, routes []models.Route) error {
	return client.UpsertRoutes(routes)
}
