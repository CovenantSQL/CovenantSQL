package api_test

import (
	"testing"

	"github.com/CovenantSQL/CovenantSQL/api"
)

func TestService(t *testing.T) {
	service := &api.Service{
		WebsocketAddr: ":8546",
	}
	service.RunServers()
	// TODO
}
