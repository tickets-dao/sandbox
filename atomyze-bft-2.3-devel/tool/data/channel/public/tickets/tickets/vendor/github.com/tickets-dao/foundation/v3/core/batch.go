package core

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/proto"
)

const batchKey = "batchTransactions"

func (cc *ChainCode) saveToBatch(
	stub shim.ChaincodeStubInterface,
	fn string, creatorSKI []byte,
	sender *proto.Address,
	args []string,
	nonce uint64,
) error {
	logger := Logger()
	txID := stub.GetTxID()
	key, err := stub.CreateCompositeKey(cc.batchPrefix, []string{txID})
	if err != nil {
		logger.Errorf("Couldn't create composite key for tx %s: %s", txID, err.Error())
		return err
	}

	txTimestamp, err := stub.GetTxTimestamp()
	if err != nil {
		logger.Errorf("Couldn't get timestamp for tx %s: %s", txID, err.Error())
		return err
	}

	data, err := pb.Marshal(&proto.PendingTx{
		Method:     fn,
		Sender:     sender,
		Args:       args,
		CreatorSKI: creatorSKI,
		Timestamp:  txTimestamp.Seconds,
		Nonce:      nonce,
	})
	if err != nil {
		logger.Errorf("Couldn't marshal transaction %s: %s", txID, err.Error())
		return err
	}
	return stub.PutState(key, data)
}

func (cc *ChainCode) loadFromBatch( //nolint:funlen
	stub shim.ChaincodeStubInterface,
	txID string,
	batchTimestamp int64,
) (*proto.PendingTx, error) {
	logger := Logger()
	key, err := stub.CreateCompositeKey(cc.batchPrefix, []string{txID})
	if err != nil {
		logger.Errorf("Couldn't create composite key for tx %s: %s", txID, err.Error())
		return nil, err
	}
	data, err := stub.GetState(key)
	if err != nil {
		logger.Errorf("Couldn't load transaction %s from state: %s", txID, err.Error())
		return nil, err
	}
	if len(data) == 0 {
		logger.Warningf("Transaction %s not found", txID)
		return nil, fmt.Errorf("transaction %s not found", txID)
	}

	defer func() {
		err = stub.DelState(key)
		if err != nil {
			logger.Errorf("Couldn't delete from state tx %s: %s", txID, err.Error())
		}
	}()

	pending := new(proto.PendingTx)
	if err = pb.Unmarshal(data, pending); err != nil {
		// возможно лежит по старому
		var args []string
		if err = json.Unmarshal(data, &args); err != nil {
			logger.Errorf("Couldn't unmarshal transaction %s: %s", txID, err.Error())
			return nil, err
		}

		creatorSKI, err := hex.DecodeString(args[1])
		if err != nil {
			return nil, err
		}

		pending = &proto.PendingTx{
			Method:     args[0],
			Args:       args[2:],
			CreatorSKI: creatorSKI,
		}
	}

	if cc.txTTL > 0 && batchTimestamp-pending.Timestamp > int64(cc.txTTL) {
		logger.Errorf("Transaction ttl expired %s", txID)
		return pending, errors.New("transaction expired")
	}

	if cc.nonceTTL != 0 {
		method, exists := cc.methods[pending.Method]
		if !exists {
			logger.Errorf("unknown method %s in tx %s", pending.Method, txID)
			return pending, fmt.Errorf("unknown method %s in tx %s", pending.Method, txID)
		}

		if !method.needsAuth {
			return pending, nil
		}

		if pending.Sender == nil {
			logger.Errorf("no sender in tx %s", txID)
			return pending, fmt.Errorf("no sender in tx %s", txID)
		}
		if err = cc.nonceCheckFn(stub, types.NewSenderFromAddr((*types.Address)(pending.Sender)), pending.Nonce); err != nil {
			logger.Errorf("incorrect tx %s nonce: %s", txID, err.Error())
			return pending, err
		}
	}

	return pending, nil
}

//nolint:funlen
func (cc *ChainCode) batchExecute(stub shim.ChaincodeStubInterface, creatorSKI string, dataIn string) peer.Response {
	logger := Logger()
	batchID := stub.GetTxID()
	btchStub := newBatchStub(stub)
	start := time.Now()
	defer func() {
		logger.Infof("batch %s elapsed time %d ms", batchID, time.Since(start).Milliseconds())
	}()
	response := proto.BatchResponse{}
	events := proto.BatchEvent{}
	var batch proto.Batch
	if err := pb.Unmarshal([]byte(dataIn), &batch); err != nil {
		logger.Errorf("Couldn't unmarshal batch %s: %s", batchID, err.Error())
		return shim.Error(err.Error())
	}

	batchTimestamp, err := stub.GetTxTimestamp()
	if err != nil {
		logger.Errorf("Couldn't get batch timestamp %s: %s", batchID, err.Error())
		return shim.Error(err.Error())
	}

	for _, txID := range batch.TxIDs {
		resp, event := cc.batchedTxExecute(btchStub, txID, batchTimestamp.Seconds)
		response.TxResponses = append(response.TxResponses, resp)
		events.Events = append(events.Events, event)
	}

	if !cc.disableSwaps {
		for _, swap := range batch.Swaps {
			response.SwapResponses = append(response.SwapResponses, swapAnswer(btchStub, creatorSKI, swap))
		}
		for _, swapKey := range batch.Keys {
			response.SwapKeyResponses = append(response.SwapKeyResponses, swapRobotDone(btchStub, creatorSKI, swapKey.Id, swapKey.Key))
		}
	}

	if !cc.disableMultiSwaps {
		for _, swap := range batch.MultiSwaps {
			response.SwapResponses = append(response.SwapResponses, multiSwapAnswer(btchStub, creatorSKI, swap))
		}
		for _, swapKey := range batch.MultiSwapsKeys {
			response.SwapKeyResponses = append(response.SwapKeyResponses, multiSwapRobotDone(btchStub, creatorSKI, swapKey.Id, swapKey.Key))
		}
	}

	if err := btchStub.Commit(); err != nil {
		logger.Errorf("Couldn't commit batch %s: %s", batchID, err.Error())
		return shim.Error(err.Error())
	}

	response.CreatedSwaps = btchStub.swaps
	response.CreatedMultiSwap = btchStub.multiSwaps

	data, err := pb.Marshal(&response)
	if err != nil {
		logger.Errorf("Couldn't marshal batch response %s: %s", batchID, err.Error())
		return shim.Error(err.Error())
	}
	eventData, err := pb.Marshal(&events)
	if err != nil {
		logger.Errorf("Couldn't marshal batch event %s: %s", batchID, err.Error())
		return shim.Error(err.Error())
	}
	if err = stub.SetEvent("batchExecute", eventData); err != nil {
		logger.Errorf("Couldn't set batch event %s: %s", batchID, err.Error())
		return shim.Error(err.Error())
	}
	return shim.Success(data)
}

type TxResponse struct {
	Method     string                    `json:"method"`
	Error      string                    `json:"error,omitempty"`
	Result     string                    `json:"result"`
	Events     map[string][]byte         `json:"events,omitempty"`
	Accounting []*proto.AccountingRecord `json:"accounting"`
}

func (cc *ChainCode) batchedTxExecute(stub *batchStub, binaryTxID []byte, batchTimestamp int64) (r *proto.TxResponse, e *proto.BatchTxEvent) {
	logger := Logger()
	start := time.Now()
	methodName := "unknown"

	txID := hex.EncodeToString(binaryTxID)
	defer func() {
		logger.Infof("batched method %s txid %s elapsed time %d ms", methodName, txID, time.Since(start).Milliseconds())
	}()

	r = &proto.TxResponse{Id: binaryTxID, Error: &proto.ResponseError{Error: "panic batchedTxExecute"}}
	e = &proto.BatchTxEvent{Id: binaryTxID, Error: &proto.ResponseError{Error: "panic batchedTxExecute"}}
	defer func() {
		if rc := recover(); rc != nil {
			logger.Criticalf("Tx %s panicked:\n%s", txID, string(debug.Stack()))
		}
	}()

	pending, err := cc.loadFromBatch(stub.ChaincodeStubInterface, txID, batchTimestamp)
	if err != nil && pending != nil {
		ee := proto.ResponseError{Error: fmt.Sprintf("function and args loading error: %s", err.Error())}
		return &proto.TxResponse{Id: binaryTxID, Method: pending.Method, Error: &ee}, &proto.BatchTxEvent{Id: binaryTxID, Method: pending.Method, Error: &ee}
	} else if err != nil {
		ee := proto.ResponseError{Error: fmt.Sprintf("function and args loading error: %s", err.Error())}
		return &proto.TxResponse{Id: binaryTxID, Error: &ee}, &proto.BatchTxEvent{Id: binaryTxID, Error: &ee}
	}

	txStub := stub.newTxStub(txID, hex.EncodeToString(pending.CreatorSKI))
	method, exists := cc.methods[pending.Method]
	if !exists {
		logger.Infof("Unknown method %s in tx %s", pending.Method, txID)
		ee := proto.ResponseError{Error: fmt.Sprintf("unknown method %s", pending.Method)}
		return &proto.TxResponse{Id: binaryTxID, Method: pending.Method, Error: &ee}, &proto.BatchTxEvent{Id: binaryTxID, Method: pending.Method, Error: &ee}
	}

	response, err := cc.callMethod(txStub, method, pending.Sender, pending.Args)
	if err != nil {
		ee := proto.ResponseError{Error: err.Error()}
		return &proto.TxResponse{Id: binaryTxID, Method: pending.Method, Error: &ee}, &proto.BatchTxEvent{Id: binaryTxID, Method: pending.Method, Error: &ee}
	}

	writes, events := txStub.Commit()

	sort.Slice(txStub.accounting, func(i, j int) bool {
		return strings.Compare(txStub.accounting[i].String(), txStub.accounting[j].String()) < 0
	})

	return &proto.TxResponse{
			Id:     binaryTxID,
			Method: pending.Method,
			Writes: writes,
		},
		&proto.BatchTxEvent{
			Id:         binaryTxID,
			Method:     pending.Method,
			Accounting: txStub.accounting,
			Events:     events,
			Result:     response,
		}
}

func batchedTxDelete(stub shim.ChaincodeStubInterface, prefix string, txID string) {
	logger := Logger()
	key, err := stub.CreateCompositeKey(prefix, []string{txID})
	if err != nil {
		logger.Errorf("Couldn't create batch key for tx %s: %s", txID, err.Error())
	}
	if err = stub.DelState(key); err != nil {
		logger.Errorf("Couldn't delete from state tx %s: %s", txID, err.Error())
	}
}
