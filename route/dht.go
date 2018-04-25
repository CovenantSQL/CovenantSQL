package route

import (
	log "github.com/sirupsen/logrus"
	"github.com/thunderdb/ThunderDB/rpc"
	"net"
)

func InitDhtServer(l net.Listener) (server *rpc.Server, err error) {
	server, err = rpc.NewServerWithService(rpc.ServiceMap{"Dht": NewDhtService()})
	if err != nil {
		log.Fatal(err)
		return
	}

	return
}
