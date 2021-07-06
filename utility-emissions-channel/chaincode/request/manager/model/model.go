// Package model : contains type definitions for
// storing data on request manager chaincode worldstate
// input/output to/from request Manager
// input/output to/from data chaincode
package model

// DataLock : defined data locks stored on request manager chaincode
type DataLock struct {
	// RequestId : request for which data is locked
	RequestId string `json:"requestId"`
	// Chaincode : name of chaincode where data is stored
	Chaincode string `json:"chaincode"`
	// Key : of locked data present a given chaincode
	Key string `json:"key"`
}
