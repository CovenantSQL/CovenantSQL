package csconn

import "github.com/CovenantSQL/CovenantSQL/proto"

type Resolver interface {
	Resolve(id *proto.RawNodeID) (string, error)
	ResolveEx(id *proto.RawNodeID) (*proto.Node, error)
}

var (
	defaultResolver Resolver
)

func RegisterResolver(resolver Resolver) {
	defaultResolver = resolver
}
