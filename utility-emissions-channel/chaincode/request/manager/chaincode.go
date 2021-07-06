package manager

import (
	"request/manager/log"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
)

// RequestManagerChaincode : manages state of a request
// also maintains locks on fabric data
// present on one chaincode installed on same channel.
// later will support multi chaincode install on same channel
type RequestManagerChaincode struct{}

// Init : store name of data chaincode
func (*RequestManagerChaincode) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

func (*RequestManagerChaincode) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	methodName, args := stub.GetFunctionAndParameters()
	method, ok := methodRegistry[methodName]
	if !ok {
		log.Errorf("[%s] [%s]", errMethodUnsupported, methodName)
		return shim.Error("not supported")
	}
	log.Infof("[Invoke] method = %s , args = %v", methodName, args)
	return method(stub, args)
}

var methodRegistry = map[string]func(stub shim.ChaincodeStubInterface, args []string) peer.Response{}

// ConfigureChaincode : configure chaincode instance.
func (*RequestManagerChaincode) ConfigureChaincode() {
	log.InitLogger(true)
}
