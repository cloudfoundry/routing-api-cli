package commands

import (
	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/models"
)

func UnRegister(client routing_api.Client, routes []models.Route) error {
	return client.DeleteRoutes(routes)
}
