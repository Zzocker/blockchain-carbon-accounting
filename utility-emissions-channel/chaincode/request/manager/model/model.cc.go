package model

// modeles.cc.go : defines all inputs/output received/returned by fabric data chaincode
// from/to request Manager chaincode

// DataChaincodeOutput : output sent by data chaincode to request manager
// aftere executing method before locking and free locks on request Manager
type DataChaincodeOutput struct {
	// Keys : list of keys that need to locked or unlocked
	// request manager chaincode
	Keys []string `json:"keys"`

	Output []DataChaincodeData `json:"output"`
}

// DataChaincodeData : output generaeted during call to data chancode
type DataChaincodeData struct {
	// Name : key of the Outputs map for StageData
	// if Name = "OUTPUT" , request manager will send data directly to client
	Name string `json:"name"`

	// Data : vale of the Outputs map for StageData
	Data []byte `json:"data"`

	// ToInclude : whether to include this Output into stage Data or not
	/* example : for getValidEmissions()
	Name : uuids
	Data : json.Marshal([]string{uuids})
	ToInclude : true
	*/
	ToInclude bool `json:"toInclude"`
}

// DataChaincodeLockInput : input send to data chaincode
// before locking the key on request manager
// invokeChaincode(args),
// args[0] : method to call
// args[1] : json.Marshal(DataChaincodeLockInput)
type DataChaincodeInput struct {
	// Keys : which need to checked before locking
	Keys []string `json:"keys"`
	// Parmas : chaincode logic specific
	Params []byte `json:"params"`
}
