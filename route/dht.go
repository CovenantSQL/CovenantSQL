package route

import (
	log "github.com/sirupsen/logrus"
	"github.com/thunderdb/ThunderDB/rpc"
	"net"
)

// InitDHTserver install DHTService payload to RPC server, also set listener
func InitDHTserver(l net.Listener) (server *rpc.Server, err error) {
	server, err = rpc.NewServerWithService(rpc.ServiceMap{"DHT": NewDHTService()})
	if err != nil {
		log.Fatal(err)
		return
	}
	server.SetListener(l)

	return
}
