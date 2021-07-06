package manager

import (
	"container/list"
	"encoding/json"
	"reflect"
	"request/manager/log"
	"request/manager/model"
	"request/mock"
	"sync"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/stretchr/testify/assert"
)

var (
	mockEmissions = []mock.Emissions{
		{
			UUID: "uuid-1",
		},
		{
			UUID: "uuid-2",
		},
		{
			UUID: "uuid-3",
		},
		{
			UUID: "uuid-4",
		},
		{
			UUID:    "uuid-5",
			TokenId: "tokenId-1",
			PartyId: "partyId-1",
		},
		{
			UUID:    "uuid-6",
			TokenId: "tokenId-2",
			PartyId: "partyId-2",
		},
	}
)

func TestLock(t *testing.T) {
	is := assert.New(t)
	// dummyUtilityEmissions chaincode
	emCCName := "UtilityEmissionsCC"
	emStub := shimtest.NewMockStub(emCCName, mock.MockEmissionsCC{})
	loadMockEmissions(emStub)
	reqStub := buildEmptyMockStub()
	reqStub.Invokables[emCCName] = emStub

	log.InitLogger(true)
	//
	reqId := "req-1"
	reqStub.MockTransactionStart("tx-1")
	toStore, toClient, err := lock(reqStub, reqId, emCCName, "getValidEmissions", model.DataChaincodeInput{
		Keys:   []string{"uuid-1", "uuid-5"},
		Params: nil,
	})
	reqStub.MockTransactionEnd("tx-1")
	// no error should be returned
	// toStore should have
	/*
		{
			validUUIDs : []string{uuid-1}
		}
	*/
	// toClient : []Emissions with uuid = uuid-1
	is.NoError(err, "lock should not return any error")
	is.NotNil(toStore)
	is.NotNil(toClient)

	wantToStore := map[string][]byte{
		"validUUIDs": {91, 34, 117, 117, 105, 100, 45, 49, 34, 93},
	}
	is.True(reflect.DeepEqual(wantToStore, toStore))

	wantToClient := []byte{91, 123, 34, 85, 85, 73, 68, 34, 58, 34, 117, 117, 105, 100, 45, 49, 34, 44, 34, 80, 97, 114, 116, 121, 73, 100, 34, 58, 34, 34, 44, 34, 84, 111, 107, 101, 110, 73, 100, 34, 58, 34, 34, 125, 93}

	is.True(reflect.DeepEqual(wantToClient, toClient))

	// check if for uuid-1 data has been locked or not
	is.Len(reqStub.State, 1)
	rawLock, ok := reqStub.State[buildLockKey(emCCName, "uuid-1")]
	is.True(ok)
	var lock model.DataLock
	err = json.Unmarshal(rawLock, &lock)
	is.NoError(err)
	is.Equal("uuid-1", lock.Key)
	is.Equal(reqId, lock.RequestId)
	is.Equal(emCCName, lock.Chaincode)
}

func TestLockOnAlreadyLocked(t *testing.T) {
	is := assert.New(t)
	// dummyUtilityEmissions chaincode
	emCCName := "UtilityEmissionsCC"
	emStub := shimtest.NewMockStub(emCCName, mock.MockEmissionsCC{})
	loadMockEmissions(emStub)
	reqStub := buildEmptyMockStub()
	reqStub.Invokables[emCCName] = emStub

	log.InitLogger(true)
	// put locks
	reqStub.State[buildLockKey(emCCName, "uuid-1")] = []byte("dummyValue")
	reqStub.State[buildLockKey(emCCName, "uuid-4")] = []byte("dummyValue")
	//
	reqId := "req-1"
	reqStub.MockTransactionStart("tx-1")
	toStore, toClient, err := lock(reqStub, reqId, emCCName, "getValidEmissions", model.DataChaincodeInput{
		Keys:   []string{"uuid-1", "uuid-4"},
		Params: nil,
	})
	reqStub.MockTransactionEnd("tx-1")

	is.Error(err)
	is.Nil(toStore)
	is.Nil(toClient)
}

func TestLockConcurrentClient(t *testing.T) {
	is := assert.New(t)
	// dummyUtilityEmissions chaincode
	emCCName := "UtilityEmissionsCC"
	emStub := shimtest.NewMockStub(emCCName, mock.MockEmissionsCC{})
	loadMockEmissions(emStub)
	reqStub := buildEmptyMockStub()
	reqStub.Invokables[emCCName] = emStub

	log.InitLogger(true)
	//
	client1reqId := "client1-req-1"
	client2reqId := "client1-req-2"
	var err1 error

	var err2 error

	wg := sync.WaitGroup{}
	// mutex is used here for simulating blockchain nature of preventing writting as key with same version
	// in fabric it will return either return MVCC error or simple error saying "key is locked"
	// but in this test one of the client will return error saying "key is locked"
	// by using mutex this no longer remain a concurnet client call
	wg.Add(2)
	mu := sync.Mutex{}
	// client : 1
	go func() {
		defer wg.Done()
		mu.Lock()
		defer mu.Unlock()
		reqStub.MockTransactionStart("tx-1")
		_, _, err1 = lock(reqStub, client1reqId, emCCName, "getValidEmissions", model.DataChaincodeInput{
			Keys:   []string{"uuid-1", "uuid-4"},
			Params: nil,
		})
		reqStub.MockTransactionEnd("tx-1")
	}()

	// client : 2
	go func() {
		defer wg.Done()
		mu.Lock()
		defer mu.Unlock()
		reqStub.MockTransactionStart("tx-2")
		_, _, err2 = lock(reqStub, client2reqId, emCCName, "getValidEmissions", model.DataChaincodeInput{
			Keys:   []string{"uuid-1", "uuid-5"},
			Params: nil,
		})
		reqStub.MockTransactionEnd("tx-2")
	}()

	wg.Wait()

	is.True((err1 == nil && err2 != nil) || (err1 != nil && err2 == nil))

	if err1 == nil {
		// uuid-1 and uuid-2 both should be locked
		_, ok := reqStub.State[buildLockKey(emCCName, "uuid-1")]
		is.True(ok)
		_, ok = reqStub.State[buildLockKey(emCCName, "uuid-4")]
		is.True(ok)
		is.Len(reqStub.State, 2)
	} else if err2 == nil {
		// only uuid-1 should be locked
		_, ok := reqStub.State[buildLockKey(emCCName, "uuid-1")]
		is.True(ok)
		is.Len(reqStub.State, 1)
	}
}

func TestUnlock(t *testing.T) {
	is := assert.New(t)
	// dummyUtilityEmissions chaincode
	emCCName := "UtilityEmissionsCC"
	emStub := shimtest.NewMockStub(emCCName, mock.MockEmissionsCC{})
	loadMockEmissions(emStub)
	reqStub := buildEmptyMockStub()
	reqStub.Invokables[emCCName] = emStub

	log.InitLogger(true)
	reqId := "reqId-1"
	// lock uuid-1 and uuid-2
	raw, _ := json.Marshal(model.DataLock{
		Chaincode: emCCName,
		Key:       "uuid-1",
		RequestId: reqId,
	})
	reqStub.State[buildLockKey(emCCName, "uuid-1")] = raw
	raw, _ = json.Marshal(model.DataLock{
		Chaincode: emCCName,
		Key:       "uuid-2",
		RequestId: reqId,
	})
	reqStub.State[buildLockKey(emCCName, "uuid-2")] = raw

	// call unlock
	ccParams, _ := json.Marshal(mock.UpdateEmissionsMintedTokenParams{
		TokenId: "tokenId-1",
		PartyId: "partyId-1",
	})
	reqStub.MockTransactionStart("tx-1")
	toStore, toClient, err := unlock(reqStub, reqId, emCCName, "UpdateEmissionsWithToken", model.DataChaincodeInput{
		Keys:   []string{"uuid-1", "uuid-2"},
		Params: ccParams,
	})
	reqStub.MockTransactionEnd("tx-1")
	is.NoError(err)
	is.Len(toStore, 0)
	is.Nil(toClient)
	// lock on uuid-1 and uuid-2 should be removed
	// utilityEmissionsChaincode should be update with partyId and tokenId
	_, ok := reqStub.State[buildLockKey(emCCName, "uuid-1")]
	is.False(ok)
	_, ok = reqStub.State[buildLockKey(emCCName, "uuid-2")]
	is.False(ok)
	is.Len(reqStub.State, 0)

	raw = emStub.State["uuid-1"]
	var emissions mock.Emissions
	json.Unmarshal(raw, &emissions)
	is.Equal("tokenId-1", emissions.TokenId)
	is.Equal("partyId-1", emissions.PartyId)

	raw = emStub.State["uuid-1"]
	emissions = mock.Emissions{}
	json.Unmarshal(raw, &emissions)
	is.Equal("tokenId-1", emissions.TokenId)
	is.Equal("partyId-1", emissions.PartyId)
}

func TestUnlockLockWithDiffReq(t *testing.T) {
	is := assert.New(t)
	// dummyUtilityEmissions chaincode
	emCCName := "UtilityEmissionsCC"
	emStub := shimtest.NewMockStub(emCCName, mock.MockEmissionsCC{})
	loadMockEmissions(emStub)
	reqStub := buildEmptyMockStub()
	reqStub.Invokables[emCCName] = emStub

	log.InitLogger(true)
	reqId := "reqId-1"
	// lock uuid-1 and uuid-2
	raw, _ := json.Marshal(model.DataLock{
		Chaincode: emCCName,
		Key:       "uuid-1",
		RequestId: reqId + "2",
	})
	reqStub.State[buildLockKey(emCCName, "uuid-1")] = raw
	raw, _ = json.Marshal(model.DataLock{
		Chaincode: emCCName,
		Key:       "uuid-2",
		RequestId: reqId,
	})
	reqStub.State[buildLockKey(emCCName, "uuid-2")] = raw

	// call unlock
	ccParams, _ := json.Marshal(mock.UpdateEmissionsMintedTokenParams{
		TokenId: "tokenId-1",
		PartyId: "partyId-1",
	})
	reqStub.MockTransactionStart("tx-1")
	// reqId is not same
	_, _, err := unlock(reqStub, reqId, emCCName, "UpdateEmissionsWithToken", model.DataChaincodeInput{
		Keys:   []string{"uuid-1", "uuid-2"},
		Params: ccParams,
	})
	reqStub.MockTransactionEnd("tx-1")
	is.Error(err)
	is.Len(reqStub.State, 2)
}

func TestUnlockFreeLock(t *testing.T) {
	is := assert.New(t)
	// dummyUtilityEmissions chaincode
	emCCName := "UtilityEmissionsCC"
	emStub := shimtest.NewMockStub(emCCName, mock.MockEmissionsCC{})
	loadMockEmissions(emStub)
	reqStub := buildEmptyMockStub()
	reqStub.Invokables[emCCName] = emStub

	log.InitLogger(true)
	reqId := "reqId-1"
	// lock uuid-1 only
	raw, _ := json.Marshal(model.DataLock{
		Chaincode: emCCName,
		Key:       "uuid-1",
		RequestId: reqId,
	})
	reqStub.State[buildLockKey(emCCName, "uuid-1")] = raw

	// call unlock
	ccParams, _ := json.Marshal(mock.UpdateEmissionsMintedTokenParams{
		TokenId: "tokenId-1",
		PartyId: "partyId-1",
	})
	reqStub.MockTransactionStart("tx-1")
	// reqId is not same
	_, _, err := unlock(reqStub, reqId, emCCName, "UpdateEmissionsWithToken", model.DataChaincodeInput{
		Keys:   []string{"uuid-1", "uuid-2"},
		Params: ccParams,
	})
	reqStub.MockTransactionEnd("tx-1")
	is.Error(err)
	is.Len(reqStub.State, 1)
}

func loadMockEmissions(emStub *shimtest.MockStub) {
	for _, em := range mockEmissions {
		raw, _ := json.Marshal(em)
		emStub.State[em.UUID] = raw
	}
}

func buildEmptyMockStub() *shimtest.MockStub {
	s := new(shimtest.MockStub)
	s.State = make(map[string][]byte)
	s.Invokables = make(map[string]*shimtest.MockStub)
	s.Keys = list.New()
	return s
}
