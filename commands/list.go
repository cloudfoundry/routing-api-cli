package commands

import (
	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/models"
)

func List(client routing_api.Client) ([]models.Route, error) {
	return client.Routes()
}
