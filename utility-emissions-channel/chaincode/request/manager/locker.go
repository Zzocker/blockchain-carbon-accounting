package manager

import (
	"encoding/json"
	"fmt"
	"request/manager/log"
	"request/manager/model"

	"github.com/hyperledger/fabric-chaincode-go/shim"
)

// locker.go : logic for completing locking and unlocking of fabric data

// 1. check lock state
// 2. invokeChiancode with free keys
// 3. place lock on return keys (after data chaincode is done with business logic)
// returns : output that need to stored for later stage
// output that will be directly sent to the client
func lock(stub shim.ChaincodeStubInterface, reqId string, dataCCName string, method string, params model.DataChaincodeInput) (map[string][]byte, []byte, error) {
	var fnTag = fmt.Sprintf("#lock::%s", dataCCName)
	log.Debugf("%s checking free lock state for %v", fnTag, params.Keys)
	for _, key := range params.Keys {
		k := buildLockKey(dataCCName, key)
		raw, err := stub.GetState(k)
		if err != nil {
			log.Errorf("%s :: %s key = %s error = %s", fnTag, errGettingState, k, err.Error())
			return nil, nil, err
		}
		if len(raw) != 0 {
			log.Errorf("%s :: %s key = %s", fnTag, errAlreadyLocked, key)
			return nil, nil, fmt.Errorf("key = %s is in locked state", key)
		}
	}
	///
	log.Debugf("%s running business logic before locking", fnTag)
	paramsRaw, _ := json.Marshal(params)
	resp := stub.InvokeChaincode(dataCCName, [][]byte{[]byte(method), paramsRaw}, "")
	if resp.Status != shim.OK {
		log.Errorf("%s %s method = %s parmas = %+v", fnTag, errInvokeChaincode, method, params)
		return nil, nil, fmt.Errorf(resp.GetMessage())
	}
	var ccOutput model.DataChaincodeOutput
	err := json.Unmarshal(resp.Payload, &ccOutput)
	if err != nil {
		log.Errorf("%s %s error = %s", fnTag, errBadDataChaincodeOutput, err.Error())
		return nil, nil, fmt.Errorf("invalid response from %s", dataCCName)
	}
	///
	log.Debugf("%s locking keys = %v ", fnTag, ccOutput.Keys)
	for _, key := range ccOutput.Keys {
		k := buildLockKey(dataCCName, key)
		raw, _ := json.Marshal(model.DataLock{
			RequestId: reqId,
			Chaincode: dataCCName,
			Key:       key,
		})
		err = stub.PutState(k, raw)
		if err != nil {
			log.Debugf("%s %s key = %s err = %s", fnTag, errPuttingState, k, err)
			return nil, nil, err
		}
	}
	log.Debugf("%s generating output for storing and sending to client", fnTag)
	outputToStore := make(map[string][]byte)
	var outputToClient []byte
	for _, output := range ccOutput.Output {
		if output.Name == "OUTPUT" && outputToClient == nil {
			outputToClient = output.Data
		} else if output.ToInclude {
			outputToStore[output.Name] = output.Data
		}
	}
	log.Debugf("%s output to client = %v", fnTag, string(outputToClient))
	if log.IsDev {
		log.Debugf("%s output to store : ", fnTag)
		for k, v := range outputToStore {
			log.Debugf("%s key = %s , value = %s", fnTag, k, string(v))
		}
	}
	log.Debugf("%s keys = %v are locked", fnTag, ccOutput.Keys)
	return outputToStore, outputToClient, nil
}

// 1. check lock state
// 2. invokeChiancode with locked keys
// 3. free lock on return keys (after data chaincode is done with business logic)
func unlock(stub shim.ChaincodeStubInterface, reqId string, dataCCName string, method string, params model.DataChaincodeInput) (map[string][]byte, []byte, error) {
	var fnTag = fmt.Sprintf("#unlock::%s", dataCCName)
	log.Debugf("%s checking locked state of keys = %v", fnTag, params.Keys)
	for _, key := range params.Keys {
		k := buildLockKey(dataCCName, key)
		raw, err := stub.GetState(k)
		if err != nil {
			log.Errorf("%s :: %s key = %s error = %s", fnTag, errGettingState, k, err.Error())
			return nil, nil, err
		}
		if len(raw) == 0 {
			log.Errorf("%s :: %s key = %s", fnTag, errFreedLock, key)
			return nil, nil, fmt.Errorf("key = %s is in free state", key)
		}
		var dataLock model.DataLock
		json.Unmarshal(raw, &dataLock)
		if dataLock.RequestId != reqId {
			log.Errorf("%s :: %s chancode = %s key = %s is locked with request_id = %s not with %s", fnTag, errReqIdMismatchOnLock, dataCCName, key, dataLock.RequestId, reqId)
			return nil, nil, fmt.Errorf("key = %s is locked with request_id = %s not with %s on cc = %s", key, dataLock.RequestId, reqId, dataCCName)
		}
	}
	///
	log.Debugf("%s running business logic method = %s before unlocking", fnTag, method)
	paramsraw, _ := json.Marshal(params)
	resp := stub.InvokeChaincode(dataCCName, [][]byte{[]byte(method), paramsraw}, "")
	if resp.Status != shim.OK {
		log.Errorf("%s error invoking method = %s : %s", fnTag, method, resp.GetMessage())
		return nil, nil, fmt.Errorf(resp.GetMessage())
	}
	var ccOutput model.DataChaincodeOutput
	err := json.Unmarshal(resp.GetPayload(), &ccOutput)
	if err != nil {
		log.Errorf("%s invalid response from chaincode : %s", fnTag, err)
		return nil, nil, fmt.Errorf("invalid response from %s", dataCCName)
	}
	///
	log.Debugf("%s unlocking keys = %s", fnTag, ccOutput.Keys)
	for _, key := range ccOutput.Keys {
		k := buildLockKey(dataCCName, key)
		err := stub.DelState(k)
		if err != nil {
			log.Debugf("%s %s kry = %s", fnTag, errDeletingState, key)
			return nil, nil, err
		}
	}
	log.Debugf("%s generating output for storing and sending to client", fnTag)
	outputToStore := make(map[string][]byte)
	var outputToClient []byte
	for _, output := range ccOutput.Output {
		if output.Name == "OUTPUT" && outputToClient == nil {
			outputToClient = output.Data
		} else if output.ToInclude {
			outputToStore[output.Name] = output.Data
		}
	}
	log.Debugf("%s output to client = %v", fnTag, string(outputToClient))
	if log.IsDev && len(outputToStore) != 0 {
		log.Debugf("%s output to store : ", fnTag)
		for k, v := range outputToStore {
			log.Debugf("%s key = %s , value = %s", fnTag, k, string(v))
		}
	}
	log.Debugf("%s keys = %v are unlocked", fnTag, ccOutput.Keys)
	return outputToStore, outputToClient, nil
}

func buildLockKey(ccName, key string) string {
	objectType := "PREFIX~CHAINCODE~KEY"
	k, _ := shim.CreateCompositeKey(objectType, []string{"LOCKER", ccName, key})
	return k
}
