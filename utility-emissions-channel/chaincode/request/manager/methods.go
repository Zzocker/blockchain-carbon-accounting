package manager

import (
	"encoding/json"
	"fmt"
	"request/manager/log"
	"request/manager/model"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
)

// methods.go : public methods supported by request manager are defined in this file

func stageUpdate(stub shim.ChaincodeStubInterface, in []byte) peer.Response {
	const fnTag = "#stageUpdate"
	var input model.StageUpdateInput
	err := json.Unmarshal(in, &input)
	if err != nil {
		log.Infof("%s %s :: %s", fnTag, errBadRequestObject, err.Error())
		return shim.Error("bad request object")
	}
	log.Debugf("%s input = %+v", fnTag, input)
	// TODO add input checks

	log.Debugf("%s get request id = %s", fnTag, input.RequestId)
	raw, err := stub.GetState(input.RequestId)
	if err != nil {
		log.Infof("%s %s key = %s", fnTag, errGettingState, input.RequestId)
		return shim.Error(err.Error())
	}
	log.Debugf("%s get caller", fnTag)
	mspId, commonName, err := getCaller(stub)
	if err != nil {
		log.Infof("%s %s", fnTag, errGettingCaller)
		return shim.Error(err.Error())
	}
	var request model.Request
	if len(raw) == 0 {
		timestamp, err := stub.GetTxTimestamp()
		if err != nil {
			log.Infof("%s error getting timestamp : %s", fnTag, err.Error())
			return shim.Error(err.Error())
		}
		log.Debugf("%s request doesn't exists, considering %s stage as first", fnTag, input.Name)
		request = model.Request{
			ID:         input.RequestId,
			State:      model.RequestStatePROCESSING,
			CallerType: input.CallerType,
			CreatedAt:  timestamp.Seconds,
			StageData:  make(map[string]*model.StageData),
		}
		if input.CallerType == model.RequestCallerTypeCLIENT {
			request.CallerID = fmt.Sprintf("%s::%s", mspId, commonName)
		} else if input.CallerType == model.RequestCallerTypeMSP {
			request.CallerID = mspId
		}

	} else {
		json.Unmarshal(raw, &request)
		if request.CallerType == model.RequestCallerTypeCLIENT && request.CallerID != fmt.Sprintf("%s::%s", mspId, commonName) {
			log.Infof("%s  wrong caller, only client can update request", fnTag)
			return shim.Error("wrong caller, only client can update request")
		} else if request.CallerType == model.RequestCallerTypeMSP && request.CallerID != mspId {
			log.Infof("%s  wrong caller, only request organization can update request", fnTag)
			return shim.Error("wrong caller, only request organization can update request")
		}
		if request.State == model.RequestStateFINISHED {
			log.Debugf("%s %s is already in FINISHED state", fnTag, input.RequestId)
			return shim.Error(fmt.Sprintf("%s is already in FINISHED state", input.RequestId))
		}
	}

	request.CurrentStageName = input.Name
	request.CurrentStageState = input.StageState
	if request.StageData[input.Name] == nil {
		request.StageData[input.Name] = &model.StageData{
			Outputs:        make(map[string]map[string][]byte),
			BlockchainData: make([]model.BlockchainData, 0),
		}
	} else {
		if request.StageData[input.Name].Outputs == nil {
			request.StageData[input.Name].Outputs = make(map[string]map[string][]byte)
		}
		if request.StageData[input.Name].BlockchainData == nil {
			request.StageData[input.Name].BlockchainData = make([]model.BlockchainData, 0)
		}
	}

	log.Debugf("%s request before : %+v", fnTag, request)
	// run fabric data locks
	log.Debugf("%s executing data lock opeartions", fnTag)
	locksOutput := make(map[string][]byte)
	if len(input.FabricDataLocks) > 0 {
		for ccName, ccInput := range input.FabricDataLocks {
			toStore, toClient, err := lock(stub, input.RequestId, ccName, ccInput.MethodName, ccInput.Input)
			if err != nil {
				return shim.Error(err.Error())
			}
			if len(toClient) > 0 {
				locksOutput[ccName] = toClient
			}
			if len(toStore) > 0 {
				request.StageData[input.Name].Outputs[ccName] = toStore
			}
		}
	}
	log.Debugf("%s executing data unlock opeartions", fnTag)
	unlockOutput := make(map[string][]byte)
	if len(input.FabricDataFree) > 0 {
		for ccName, ccInput := range input.FabricDataFree {
			toStore, toClient, err := unlock(stub, input.RequestId, ccName, ccInput.MethodName, ccInput.Input)
			if err != nil {
				return shim.Error(err.Error())
			}
			if len(toClient) > 0 {
				unlockOutput[ccName] = toClient
			}
			if len(toStore) > 0 {
				request.StageData[input.Name].Outputs[ccName] = toStore
			}
		}
	}
	log.Debugf("%s updating stage blockchain data", fnTag)
	if len(input.BlockchainData) > 0 {
		request.StageData[input.Name].BlockchainData = append(request.StageData[input.Name].BlockchainData, input.BlockchainData...)
	}

	if input.IsLast && input.StageState == "FINISHED" {
		log.Infof("%s requestId = %s is finished", fnTag, input.RequestId)
		request.State = model.RequestStateFINISHED
	}
	log.Debugf("%s putting request", fnTag)
	raw, _ = json.Marshal(request)
	err = stub.PutState(input.RequestId, raw)
	if err != nil {
		log.Infof("%s %s key = %s", fnTag, errPuttingState, input.RequestId)
		return shim.Error(err.Error())
	}
	//
	log.Debugf("%s request after : %+v", fnTag, string(raw))
	out := model.StageUpdateOutput{
		FabricDataLocks: locksOutput,
		FabricDataFree:  unlockOutput,
	}
	raw, _ = json.Marshal(out)
	return shim.Success(raw)
}
