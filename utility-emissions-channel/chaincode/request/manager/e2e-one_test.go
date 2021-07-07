package manager

import (
	"encoding/json"
	"reflect"
	"request/manager/model"
	"request/mock"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/stretchr/testify/assert"
)

func TestE2E(t *testing.T) {
	is := assert.New(t)
	emCCName := "UtilityEmissionsChaincode"
	emStub := shimtest.NewMockStub(emCCName, new(mock.MockEmissionsCC))
	loadMockEmissions(emStub)

	reqCCName := "RequestManager"
	cc := new(RequestManagerChaincode)
	cc.ConfigureChaincode()
	reqStub := shimtest.NewMockStub(reqCCName, cc)
	reqStub.Invokables[emCCName] = emStub

	reqId := "req-id-1"
	// first stage
	// 1 . lock
	// set caller to user1
	t.Run("Stage-1::GET_VALID_EMISSIONS", func(t *testing.T) {
		setCaller(reqStub, user1)
		stage1Input := model.StageUpdateInput{
			RequestId: reqId,
			Name:      "GET_VALID_EMISSIONS",
			FabricDataLocks: map[string]model.RequestDataChaincodeInput{
				emCCName: {
					MethodName: "getValidEmissions",
					Input: model.DataChaincodeInput{
						Keys:   []string{"uuid-1", "uuid-2", "uuid-5"},
						Params: nil,
					},
				},
			},
			StageState: "FINISHED",
			CallerType: model.RequestCallerTypeCLIENT,
		}
		raw, _ := json.Marshal(stage1Input)
		resp := reqStub.MockInvoke("tx-1", [][]byte{[]byte("stageUpdate"), raw})

		is.Equal(shim.OK, int(resp.GetStatus()), resp.GetMessage())
		// check resp Playload
		is.NotNil(resp.GetPayload())

		var output model.StageUpdateOutput
		err := json.Unmarshal(resp.GetPayload(), &output)
		is.NoError(err)

		raw, ok := output.FabricDataLocks[emCCName]
		is.True(ok)
		var lockedEmissions []mock.Emissions
		err = json.Unmarshal(raw, &lockedEmissions)
		is.NoError(err)
		is.Len(lockedEmissions, 2)

		is.Len(output.FabricDataFree, 0)

		// check request update
		raw, ok = reqStub.State[reqId]
		is.True(ok)
		var request model.Request
		err = json.Unmarshal(raw, &request)
		is.NoError(err)
		is.Equal("auditor1::user1", request.CallerID)
		raw, ok = request.StageData["GET_VALID_EMISSIONS"].Outputs[emCCName]["validUUIDs"]
		is.True(ok)
		var validUUIDs []string
		err = json.Unmarshal(raw, &validUUIDs)
		is.NoError(err)
		is.True(reflect.DeepEqual(validUUIDs, []string{"uuid-1", "uuid-2"}))

		_, ok = reqStub.State[buildLockKey(emCCName, "uuid-1")]
		is.True(ok)

		_, ok = reqStub.State[buildLockKey(emCCName, "uuid-2")]
		is.True(ok)

		is.Equal("GET_VALID_EMISSIONS", request.CurrentStageName)
	})

	t.Run("Stage-2::BadObject", func(t *testing.T) {
		resp := reqStub.MockInvoke("tx-1", [][]byte{[]byte("stageUpdate"), {}})
		is.Equal(shim.ERROR, int(resp.GetStatus()))
	})

	t.Run("Stage-2::EmptyCallerCert", func(t *testing.T) {
		reqStub.Creator = nil
		stage1Input := model.StageUpdateInput{
			RequestId:  reqId,
			Name:       "TOKEN_MINTING",
			StageState: "FINISHED",
			BlockchainData: []model.BlockchainData{
				{
					Network:         "Ethereum",
					ContractAddress: "0x5757fe.....",
					KeysCreated: map[string]string{
						"tokenId": "0x77576576",
					},
				},
			},
		}
		raw, _ := json.Marshal(stage1Input)
		resp := reqStub.MockInvoke("tx-1", [][]byte{[]byte("stageUpdate"), raw})
		is.Equal(shim.ERROR, int(resp.GetStatus()))
	})

	t.Run("Stage-2::WithOtherCaller", func(t *testing.T) {
		setCaller(reqStub, auditor2admin)
		stage1Input := model.StageUpdateInput{
			RequestId:  reqId,
			Name:       "TOKEN_MINTING",
			StageState: "FINISHED",
			BlockchainData: []model.BlockchainData{
				{
					Network:         "Ethereum",
					ContractAddress: "0x5757fe.....",
					KeysCreated: map[string]string{
						"tokenId": "0x77576576",
					},
				},
			},
		}
		raw, _ := json.Marshal(stage1Input)
		resp := reqStub.MockInvoke("tx-1", [][]byte{[]byte("stageUpdate"), raw})
		is.Equal(shim.ERROR, int(resp.GetStatus()))
	})

	t.Run("Stage-2::TOKEN_MINTING", func(t *testing.T) {
		setCaller(reqStub, user1)
		stage1Input := model.StageUpdateInput{
			RequestId:  reqId,
			Name:       "TOKEN_MINTING",
			StageState: "FINISHED",
			BlockchainData: []model.BlockchainData{
				{
					Network:         "Ethereum",
					ContractAddress: "0x5757fe",
					KeysCreated: map[string]string{
						"tokenId": "0x77576576",
					},
				},
			},
		}
		raw, _ := json.Marshal(stage1Input)
		resp := reqStub.MockInvoke("tx-1", [][]byte{[]byte("stageUpdate"), raw})
		is.Equal(shim.OK, int(resp.GetStatus()))
		var output model.StageUpdateOutput
		err := json.Unmarshal(resp.GetPayload(), &output)
		is.NoError(err)

		is.Len(output.FabricDataFree, 0)
		is.Len(output.FabricDataLocks, 0)

		// check on request updates
		raw, ok := reqStub.State[reqId]
		is.True(ok)
		var request model.Request
		err = json.Unmarshal(raw, &request)
		is.NoError(err)

		data, ok := request.StageData["TOKEN_MINTING"]
		is.True(ok)
		is.Len(data.BlockchainData, 1)
		is.Equal("Ethereum", data.BlockchainData[0].Network)
		is.Equal("0x5757fe", data.BlockchainData[0].ContractAddress)
		tokenId, ok := data.BlockchainData[0].KeysCreated["tokenId"]
		is.True(ok)
		is.Equal("0x77576576", tokenId)

		is.Equal("TOKEN_MINTING", request.CurrentStageName)
		is.Equal("FINISHED", request.CurrentStageState)
	})

	t.Run("Stage-3::UPDATE_TOKEN_ID", func(t *testing.T) {
		setCaller(reqStub, user1)
		ccParams, _ := json.Marshal(mock.UpdateEmissionsMintedTokenParams{
			TokenId: "tokenId-1",
			PartyId: "partyId-1",
		})
		stage1Input := model.StageUpdateInput{
			RequestId:  reqId,
			Name:       "UPDATE_TOKEN_ID",
			StageState: "FINISHED",
			IsLast:     true,
			FabricDataFree: map[string]model.RequestDataChaincodeInput{
				emCCName: {
					MethodName: "UpdateEmissionsWithToken",
					Input: model.DataChaincodeInput{
						Keys:   []string{"uuid-1", "uuid-2"},
						Params: ccParams,
					},
				},
			},
		}
		raw, _ := json.Marshal(stage1Input)
		resp := reqStub.MockInvoke("tx-1", [][]byte{[]byte("stageUpdate"), raw})
		is.Equal(shim.OK, int(resp.GetStatus()))
		var output model.StageUpdateOutput
		err := json.Unmarshal(resp.GetPayload(), &output)
		is.NoError(err)

		is.Len(output.FabricDataFree, 0)
		is.Len(output.FabricDataLocks, 0)

		// check on request updates
		raw, ok := reqStub.State[reqId]
		is.True(ok)
		var request model.Request
		err = json.Unmarshal(raw, &request)
		is.NoError(err)

		_, ok = request.StageData["UPDATE_TOKEN_ID"]
		is.True(ok)

		is.Equal("UPDATE_TOKEN_ID", request.CurrentStageName)
		is.Equal("FINISHED", request.CurrentStageState)
		is.Equal(model.RequestStateFINISHED, request.State)

		_, ok = reqStub.State[buildLockKey(emCCName, "uuid-1")]
		is.False(ok)

		_, ok = reqStub.State[buildLockKey(emCCName, "uuid-2")]
		is.False(ok)
	})

	t.Run("Stage-3::CallingAfterRequestIsFinished", func(t *testing.T) {
		setCaller(reqStub, user1)
		ccParams, _ := json.Marshal(mock.UpdateEmissionsMintedTokenParams{
			TokenId: "tokenId-1",
			PartyId: "partyId-1",
		})
		stage1Input := model.StageUpdateInput{
			RequestId:  reqId,
			Name:       "UPDATE_TOKEN_ID",
			StageState: "FINISHED",
			IsLast:     true,
			FabricDataFree: map[string]model.RequestDataChaincodeInput{
				emCCName: {
					MethodName: "UpdateEmissionsWithToken",
					Input: model.DataChaincodeInput{
						Keys:   []string{"uuid-1", "uuid-2"},
						Params: ccParams,
					},
				},
			},
		}
		raw, _ := json.Marshal(stage1Input)
		resp := reqStub.MockInvoke("tx-1", [][]byte{[]byte("stageUpdate"), raw})
		is.Equal(shim.ERROR, int(resp.GetStatus()))
	})
}
