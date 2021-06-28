package manager

import (
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"request/manager/log"
)

// RequestManagerChaincode : manages state of a request
// also maintains locks on fabric data
// present on different chaincode installed on same channel.
type RequestManagerChaincode struct{}

func (*RequestManagerChaincode) Init(stub shim.ChaincodeStubInterface) peer.Response {
	channelName := stub.GetChannelID()
	log.Infof("Initializing Request Manager Chaincode on %s", channelName)
	if err := stub.PutState(ccChannelKey, []byte(channelName)); err != nil {
		log.Errorf("[%s] [%s] %s", errPuttingState, ccChannelKey, err.Error())
		return shim.Error(err.Error())
	}
	log.Info("Chaincode initialized")
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
