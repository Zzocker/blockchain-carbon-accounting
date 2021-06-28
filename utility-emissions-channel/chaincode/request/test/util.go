package test

import (
	"request/manager"

	"github.com/hyperledger/fabric-chaincode-go/shimtest"
)

func prepareManager() *shimtest.MockStub {
	cc := new(manager.RequestManagerChaincode)
	cc.ConfigureChaincode()

	return shimtest.NewMockStub("Manager", cc)
}
