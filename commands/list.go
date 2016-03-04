package commands

import (
	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/db"
)

func List(client routing_api.Client) ([]db.Route, error) {
	return client.Routes()
}
