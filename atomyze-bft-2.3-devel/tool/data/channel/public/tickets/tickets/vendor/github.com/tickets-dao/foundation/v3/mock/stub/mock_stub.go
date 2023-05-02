/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package stub mocked_atomyze provides APIs for the chaincode to access its state
// variables, transaction context and call other chaincodes.
package stub

import (
	"container/list"
	"encoding/pem"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/hyperledger/fabric-protos-go/msp"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/common/util"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"github.com/tickets-dao/foundation/v3/core/types/big"
)

// Logger for the shim package.
var mockLogger = logging.MustGetLogger("mock")

const ErrFuncNotImplemented = "function %s is not implemented"

// Stub is an implementation of ChaincodeStubInterface for unit testing chaincode.
// Use this instead of ChaincodeStub in your chaincode's unit test calls to Init or Invoke.
type Stub struct {
	// A pointer back to the chaincode that will invoke this, set by constructor.
	// If a peer calls this stub, the chaincode will be invoked from here.
	cc         shim.Chaincode
	args       [][]byte          // arguments the stub was called with
	Name       string            // A nice name that can be used for logging
	State      map[string][]byte // State keeps name value pairs
	Keys       *list.List        // Keys stores the list of mapped values in lexical order registered list of other Stub chaincodes that can be called from this Stub
	Invokables map[string]*Stub
	// TODO if a chaincode uses recursion this may need to be a stack of TxIDs or possibly a reference counting map
	TxID                   string // stores a transaction uuid while being Invoked / Deployed
	TxTimestamp            *timestamp.Timestamp
	signedProposal         *pb.SignedProposal // mocked signedProposal
	ChannelID              string             // stores a channel ID of the proposal
	PvtState               map[string]map[string][]byte
	EndorsementPolicies    map[string]map[string][]byte // stores per-key endorsement policy, first map index is the collection, second map index is the key
	ChaincodeEventsChannel chan *pb.ChaincodeEvent      // channel to store ChaincodeEvents
	Decorations            map[string][]byte
	creator                []byte
}

func (stub *Stub) GetTxID() string {
	return stub.TxID
}

func (stub *Stub) GetChannelID() string {
	return stub.ChannelID
}

func (stub *Stub) GetArgs() [][]byte {
	return stub.args
}

func (stub *Stub) GetStringArgs() []string {
	args := stub.GetArgs()
	strargs := make([]string, 0, len(args))
	for _, barg := range args {
		strargs = append(strargs, string(barg))
	}
	return strargs
}

func (stub *Stub) GetFunctionAndParameters() (function string, params []string) {
	allArgs := stub.GetStringArgs()
	function = ""
	params = []string{}
	if len(allArgs) >= 1 {
		function = allArgs[0]
		params = allArgs[1:]
	}
	return
}

// MockTransactionStart is used to indicate to a chaincode that it is part of a transaction.
// This is important when chaincodes invoke each other.
// Stub doesn't support concurrent transactions at present.
func (stub *Stub) MockTransactionStart(txID string) {
	stub.TxID = txID
	stub.setSignedProposal(&pb.SignedProposal{})
	stub.setTxTimestamp(util.CreateUtcTimestamp())
}

// MockTransactionEnd ends a mocked transaction, clearing the UUID.
func (stub *Stub) MockTransactionEnd(_ string) { // uuid
	stub.signedProposal = nil
	stub.TxID = ""
}

// MockPeerChaincode registers a peer chaincode with this Stub
// invokeableChaincodeName is the name or hash of the peer
// otherStub is a Stub of the peer, already intialised
func (stub *Stub) MockPeerChaincode(invokableChaincodeName string, otherStub *Stub) {
	stub.Invokables[invokableChaincodeName] = otherStub
}

func (stub *Stub) MockPeerChaincodeWithChannel(invokableChaincodeName string, otherStub *Stub, channel string) {
	// Internally we use chaincode name as a composite name
	if channel != "" {
		invokableChaincodeName = invokableChaincodeName + "/" + channel
	}

	stub.Invokables[invokableChaincodeName] = otherStub
}

// MockInit initializes this chaincode,  also starts and ends a transaction.
func (stub *Stub) MockInit(uuid string, args [][]byte) pb.Response {
	stub.args = args
	stub.MockTransactionStart(uuid)
	res := stub.cc.Init(stub)
	stub.MockTransactionEnd(uuid)
	return res
}

// MockInvoke invokes this chaincode, also starts and ends a transaction.
func (stub *Stub) MockInvoke(uuid string, args [][]byte) pb.Response {
	stub.args = args
	stub.MockTransactionStart(uuid)
	res := stub.cc.Invoke(stub)
	stub.MockTransactionEnd(uuid)
	return res
}

func (stub *Stub) GetDecorations() map[string][]byte {
	return stub.Decorations
}

// MockInvokeWithSignedProposal invokes this chaincode, also starts and ends a transaction.
func (stub *Stub) MockInvokeWithSignedProposal(uuid string, args [][]byte, sp *pb.SignedProposal) pb.Response {
	stub.args = args
	stub.MockTransactionStart(uuid)
	stub.signedProposal = sp
	res := stub.cc.Invoke(stub)
	stub.MockTransactionEnd(uuid)
	return res
}

func (stub *Stub) GetPrivateData(collection string, key string) ([]byte, error) {
	m, in := stub.PvtState[collection]

	if !in {
		return nil, nil
	}

	return m[key], nil
}

func (stub *Stub) GetPrivateDataHash(_, _ string) ([]byte, error) { // collection, key
	return nil, errors.Errorf(ErrFuncNotImplemented, "GetPrivateDataHash")
}

func (stub *Stub) PutPrivateData(collection string, key string, value []byte) error {
	m, in := stub.PvtState[collection]
	if !in {
		stub.PvtState[collection] = make(map[string][]byte)
		m = stub.PvtState[collection]
	}

	m[key] = value

	return nil
}

func (stub *Stub) DelPrivateData(_, _ string) error { // collection, key
	return fmt.Errorf(ErrFuncNotImplemented, "DelPrivateData")
}

func (stub *Stub) GetPrivateDataByRange(_, _, _ string) (shim.StateQueryIteratorInterface, error) { // collection, startKey, endKey
	return nil, fmt.Errorf(ErrFuncNotImplemented, "GetPrivateDataByRange")
}

func (stub *Stub) GetPrivateDataByPartialCompositeKey(_, _ string, _ []string) (shim.StateQueryIteratorInterface, error) { // collection, objectType, attributes
	return nil, fmt.Errorf(ErrFuncNotImplemented, "GetPrivateDataByPartialCompositeKey")
}

func (stub *Stub) GetPrivateDataQueryResult(_, _ string) (shim.StateQueryIteratorInterface, error) { // collection, query
	// Not implemented since the mock engine does not have a query engine.
	// However, a very simple query engine that supports string matching
	// could be implemented to test that the framework supports queries
	return nil, fmt.Errorf(ErrFuncNotImplemented, "GetPrivateDataQueryResult")
}

func (stub *Stub) PurgePrivateData(_, _ string) error { // collection, key
	// TODO implement me
	return fmt.Errorf(ErrFuncNotImplemented, "PurgePrivateData")
}

// GetState retrieves the value for a given key from the Ledger
func (stub *Stub) GetState(key string) ([]byte, error) {
	value := stub.State[key]
	mockLogger.Debug("Stub", stub.Name, "Getting", key, value)
	return value, nil
}

// PutState writes the specified `value` and `key` into the Ledger.
func (stub *Stub) PutState(key string, value []byte) error {
	return stub.putState(key, value, true)
}

// PutState writes the specified `value` and `key` into the Ledger.
func (stub *Stub) putState(key string, value []byte, checkTxID bool) error {
	if checkTxID && stub.TxID == "" {
		err := errors.New("cannot PutState without a transactions - call stub.MockTransactionStart()?")
		mockLogger.Errorf("%+v", err)
		return err
	}

	// If the value is nil or empty, delete the key
	if len(value) == 0 {
		mockLogger.Debug("Stub", stub.Name, "PutState called, but value is nil or empty. Delete ", key)
		return stub.DelState(key)
	}

	mockLogger.Debug("Stub", stub.Name, "Putting", key, value)
	stub.State[key] = value

	// insert key into ordered list of keys
OuterLoop:
	for elem := stub.Keys.Front(); elem != nil; elem = elem.Next() {
		elemValue, ok := elem.Value.(string)
		if !ok {
			err := errors.New("cannot assertion elem to string")
			mockLogger.Errorf("%+v", err)
			return err
		}
		comp := strings.Compare(key, elemValue)
		mockLogger.Debug("Stub", stub.Name, "Compared", key, elemValue, " and got ", comp)
		switch {
		case comp < 0:
			stub.Keys.InsertBefore(key, elem)
			mockLogger.Debug("Stub", stub.Name, "Key", key, " inserted before", elem.Value)
			break OuterLoop
		case comp == 0:
			mockLogger.Debug("Stub", stub.Name, "Key", key, "already in State")
			break OuterLoop
		default:
			if elem.Next() == nil {
				stub.Keys.PushBack(key)
				mockLogger.Debug("Stub", stub.Name, "Key", key, "appended")
				break OuterLoop
			}
		}
	}

	// special case for empty Keys list
	if stub.Keys.Len() == 0 {
		stub.Keys.PushFront(key)
		mockLogger.Debug("Stub", stub.Name, "Key", key, "is first element in list")
	}

	return nil
}

func (stub *Stub) PutBalanceToState(key string, balance *big.Int) error {
	value := balance.Bytes()
	return stub.putState(key, value, false)
}

// DelState removes the specified `key` and its value from the Ledger.
func (stub *Stub) DelState(key string) error {
	mockLogger.Debug("Stub", stub.Name, "Deleting", key, stub.State[key])
	delete(stub.State, key)

	for elem := stub.Keys.Front(); elem != nil; elem = elem.Next() {
		el, ok := elem.Value.(string)
		if !ok {
			return errors.New("type assertion failed")
		}
		if strings.Compare(key, el) == 0 {
			stub.Keys.Remove(elem)
		}
	}

	return nil
}

func (stub *Stub) GetStateByRange(startKey, endKey string) (shim.StateQueryIteratorInterface, error) {
	if err := validateSimpleKeys(startKey, endKey); err != nil {
		return nil, err
	}
	return NewMockStateRangeQueryIterator(stub, startKey, endKey), nil
}

// GetQueryResult function can be invoked by a chaincode to perform a
// rich query against state database.  Only supported by state database implementations
// that support rich query.  The query string is in the syntax of the underlying
// state database. An iterator is returned which can be used to iterate (next) over
// the query result set
func (stub *Stub) GetQueryResult(_ string) (shim.StateQueryIteratorInterface, error) { // query
	// Not implemented since the mock engine does not have a query engine.
	// However, a very simple query engine that supports string matching
	// could be implemented to test that the framework supports queries
	return nil, fmt.Errorf(ErrFuncNotImplemented, "GetQueryResult")
}

// GetHistoryForKey function can be invoked by a chaincode to return a history of
// key values across time. GetHistoryForKey is intended to be used for read-only queries.
func (stub *Stub) GetHistoryForKey(_ string) (shim.HistoryQueryIteratorInterface, error) { // key
	return nil, fmt.Errorf(ErrFuncNotImplemented, "GetHistoryForKey")
}

// GetStateByPartialCompositeKey function can be invoked by a chaincode to query the
// state based on a given partial composite key. This function returns an
// iterator which can be used to iterate over all composite keys whose prefix
// matches the given partial composite key. This function should be used only for
// a partial composite key. For a full composite key, an iter with empty response
// would be returned.
func (stub *Stub) GetStateByPartialCompositeKey(objectType string, attributes []string) (shim.StateQueryIteratorInterface, error) {
	partialCompositeKey, err := stub.CreateCompositeKey(objectType, attributes)
	if err != nil {
		return nil, err
	}
	return NewMockStateRangeQueryIterator(stub, partialCompositeKey, partialCompositeKey+string(utf8.MaxRune)), nil
}

// CreateCompositeKey combines the list of attributes
// to form a composite key.
func (stub *Stub) CreateCompositeKey(objectType string, attributes []string) (string, error) {
	return createCompositeKey(objectType, attributes)
}

// SplitCompositeKey splits the composite key into attributes
// on which the composite key was formed.
func (stub *Stub) SplitCompositeKey(compositeKey string) (string, []string, error) {
	return splitCompositeKey(compositeKey)
}

func (stub *Stub) GetStateByRangeWithPagination(
	_, _ string, // startKey, endKey
	_ int32, // pageSize
	_ string, // bookmark
) (shim.StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {
	return nil, nil, fmt.Errorf(ErrFuncNotImplemented, "GetStateByRangeWithPagination")
}

func (stub *Stub) GetStateByPartialCompositeKeyWithPagination(
	_ string, // objectType
	_ []string, // keys
	_ int32, // pageSize
	_ string, // bookmark
) (shim.StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {
	return nil, nil, fmt.Errorf(ErrFuncNotImplemented, "GetStateByPartialCompositeKeyWithPagination")
}

func (stub *Stub) GetQueryResultWithPagination(
	_ string, // query
	_ int32, // pageSize
	_ string, // bookmark
) (shim.StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {
	return nil, nil, fmt.Errorf(ErrFuncNotImplemented, "GetQueryResultWithPagination")
}

// InvokeChaincode calls a peered chaincode.
// E.g. stub1.InvokeChaincode("stub2Hash", funcArgs, channel)
// Before calling this make sure to create another Stub stub2, call stub2.MockInit(uuid, func, args)
// and register it with stub1 by calling stub1.MockPeerChaincode("stub2Hash", stub2)
func (stub *Stub) InvokeChaincode(chaincodeName string, args [][]byte, channel string) pb.Response {
	// Internally we use chaincode name as a composite name
	if channel != "" {
		chaincodeName = chaincodeName + "/" + channel
	}
	// TODO "args" here should possibly be a serialized pb.ChaincodeInput
	otherStub := stub.Invokables[chaincodeName]
	mockLogger.Debug("Stub", stub.Name, "Invoking peer chaincode", otherStub.Name, args)
	//	function, strings := getFuncArgs(args)
	res := otherStub.MockInvoke(stub.TxID, args)
	mockLogger.Debug("Stub", stub.Name, "Invoked peer chaincode", otherStub.Name, "got", fmt.Sprintf("%+v", res))
	return res
}

// SetCreator sets creator
func (stub *Stub) SetCreator(creator []byte) {
	stub.creator = creator
}

func (stub *Stub) SetCreatorCert(creatorMSP string, creatorCert []byte) error {
	pemblock := &pem.Block{Type: "CERTIFICATE", Bytes: creatorCert}
	pemBytes := pem.EncodeToMemory(pemblock)
	if pemBytes == nil {
		return errors.New("encoding of identity failed")
	}

	creator := &msp.SerializedIdentity{Mspid: creatorMSP, IdBytes: pemBytes}
	marshaledIdentity, err := proto.Marshal(creator)
	if err != nil {
		return err
	}
	stub.creator = marshaledIdentity
	return nil
}

// GetCreator returns creator.
func (stub *Stub) GetCreator() ([]byte, error) {
	return stub.creator, nil
}

// GetTransient returns transient. Not implemented
func (stub *Stub) GetTransient() (map[string][]byte, error) {
	return nil, fmt.Errorf(ErrFuncNotImplemented, "GetTransient")
}

// GetBinding returns binding. Not implemented
func (stub *Stub) GetBinding() ([]byte, error) {
	return nil, fmt.Errorf(ErrFuncNotImplemented, "GetBinding")
}

// GetSignedProposal returns proposal. Not implemented
func (stub *Stub) GetSignedProposal() (*pb.SignedProposal, error) {
	return stub.signedProposal, nil
}

func (stub *Stub) setSignedProposal(sp *pb.SignedProposal) {
	stub.signedProposal = sp
}

// GetArgsSlice returns args slice. Not implemented
func (stub *Stub) GetArgsSlice() ([]byte, error) {
	return nil, fmt.Errorf(ErrFuncNotImplemented, "GetArgsSlice")
}

func (stub *Stub) setTxTimestamp(time *timestamp.Timestamp) {
	stub.TxTimestamp = time
}

func (stub *Stub) GetTxTimestamp() (*timestamp.Timestamp, error) {
	if stub.TxTimestamp == nil {
		return nil, errors.New("timestamp was not set")
	}
	return stub.TxTimestamp, nil
}

func (stub *Stub) SetEvent(name string, payload []byte) error {
	stub.ChaincodeEventsChannel <- &pb.ChaincodeEvent{EventName: name, Payload: payload}
	return nil
}

func (stub *Stub) SetStateValidationParameter(key string, ep []byte) error {
	return stub.SetPrivateDataValidationParameter("", key, ep)
}

func (stub *Stub) GetStateValidationParameter(key string) ([]byte, error) {
	return stub.GetPrivateDataValidationParameter("", key)
}

func (stub *Stub) SetPrivateDataValidationParameter(collection, key string, ep []byte) error {
	m, in := stub.EndorsementPolicies[collection]
	if !in {
		stub.EndorsementPolicies[collection] = make(map[string][]byte)
		m = stub.EndorsementPolicies[collection]
	}

	m[key] = ep
	return nil
}

func (stub *Stub) GetPrivateDataValidationParameter(collection, key string) ([]byte, error) {
	m, in := stub.EndorsementPolicies[collection]

	if !in {
		return nil, nil
	}

	return m[key], nil
}

// NewMockStub - Constructor to initialize the internal State map
func NewMockStub(name string, cc shim.Chaincode) *Stub {
	mockLogger.Debug("Stub(", name, cc, ")")
	s := new(Stub)
	s.Name = name
	s.cc = cc
	s.State = make(map[string][]byte)
	s.PvtState = make(map[string]map[string][]byte)
	s.EndorsementPolicies = make(map[string]map[string][]byte)
	s.Invokables = make(map[string]*Stub)
	s.Keys = list.New()
	s.ChaincodeEventsChannel = make(chan *pb.ChaincodeEvent, 100) //nolint:gomnd    // define large capacity for non-blocking setEvent calls.
	s.Decorations = make(map[string][]byte)

	return s
}

/*****************************
 Range Query Iterator
*****************************/

type StateRangeQueryIterator struct {
	Closed   bool
	Stub     *Stub
	StartKey string
	EndKey   string
	Current  *list.Element
}

// HasNext returns true if the range query iterator contains additional keys
// and values.
func (iter *StateRangeQueryIterator) HasNext() bool {
	if iter.Closed {
		// previously called Close()
		mockLogger.Debug("HasNext() but already closed")
		return false
	}

	if iter.Current == nil {
		mockLogger.Error("HasNext() couldn't get Current")
		return false
	}

	current := iter.Current
	for current != nil {
		// if this is an open-ended query for all keys, return true
		if iter.StartKey == "" && iter.EndKey == "" {
			return true
		}
		curStr, _ := current.Value.(string)
		comp1 := strings.Compare(curStr, iter.StartKey)
		comp2 := strings.Compare(curStr, iter.EndKey)
		if comp1 >= 0 {
			if comp2 < 0 {
				mockLogger.Debug("HasNext() got next")
				return true
			}

			mockLogger.Debug("HasNext() but no next")
			return false
		}
		current = current.Next()
	}

	// we've reached the end of the underlying values
	mockLogger.Debug("HasNext() but no next")
	return false
}

// Next returns the next key and value in the range query iterator.
func (iter *StateRangeQueryIterator) Next() (*queryresult.KV, error) {
	if iter.Closed {
		err := errors.New("StateRangeQueryIterator.Next() called after Close()")
		mockLogger.Errorf("%+v", err)
		return nil, err
	}

	if !iter.HasNext() {
		err := errors.New("StateRangeQueryIterator.Next() called when it does not HaveNext()")
		mockLogger.Errorf("%+v", err)
		return nil, err
	}

	for iter.Current != nil {
		curStr, _ := iter.Current.Value.(string)
		comp1 := strings.Compare(curStr, iter.StartKey)
		comp2 := strings.Compare(curStr, iter.EndKey)
		// compare to start and end keys. or, if this is an open-ended query for
		// all keys, it should always return the key and value
		if (comp1 >= 0 && comp2 < 0) || (iter.StartKey == "" && iter.EndKey == "") {
			key, _ := iter.Current.Value.(string)

			value, err := iter.Stub.GetState(key)
			iter.Current = iter.Current.Next()
			return &queryresult.KV{Key: key, Value: value}, err
		}
		iter.Current = iter.Current.Next()
	}
	err := errors.New("StateRangeQueryIterator.Next() went past end of range")
	mockLogger.Errorf("%+v", err)
	return nil, err
}

// Close closes the range query iterator. This should be called when done
// reading from the iterator to free up resources.
func (iter *StateRangeQueryIterator) Close() error {
	if iter.Closed {
		err := errors.New("StateRangeQueryIterator.Close() called after Close()")
		mockLogger.Errorf("%+v", err)
		return err
	}

	iter.Closed = true
	return nil
}

func (iter *StateRangeQueryIterator) Print() {
	mockLogger.Debug("StateRangeQueryIterator {")
	mockLogger.Debug("Closed?", iter.Closed)
	mockLogger.Debug("Stub", iter.Stub)
	mockLogger.Debug("StartKey", iter.StartKey)
	mockLogger.Debug("EndKey", iter.EndKey)
	mockLogger.Debug("Current", iter.Current)
	mockLogger.Debug("HasNext?", iter.HasNext())
	mockLogger.Debug("}")
}

func NewMockStateRangeQueryIterator(stub *Stub, startKey string, endKey string) *StateRangeQueryIterator {
	mockLogger.Debug("NewMockStateRangeQueryIterator(", stub, startKey, endKey, ")")
	iter := new(StateRangeQueryIterator)
	iter.Closed = false
	iter.Stub = stub
	iter.StartKey = startKey
	iter.EndKey = endKey
	iter.Current = stub.Keys.Front()

	iter.Print()

	return iter
}

const (
	minUnicodeRuneValue   = 0            // U+0000
	maxUnicodeRuneValue   = utf8.MaxRune // U+10FFFF - maximum (and unallocated) code point
	compositeKeyNamespace = "\x00"
	// emptyKeySubstitute    = "\x01"
)

func validateSimpleKeys(simpleKeys ...string) error {
	for _, key := range simpleKeys {
		if len(key) > 0 && key[0] == compositeKeyNamespace[0] {
			return errors.Errorf(`first character of the key [%s] contains a null character which is not allowed`, key)
		}
	}
	return nil
}

func createCompositeKey(objectType string, attributes []string) (string, error) {
	if err := validateCompositeKeyAttribute(objectType); err != nil {
		return "", err
	}
	ck := compositeKeyNamespace + objectType + string(rune(minUnicodeRuneValue))
	for _, att := range attributes {
		if err := validateCompositeKeyAttribute(att); err != nil {
			return "", err
		}
		ck += att + string(rune(minUnicodeRuneValue))
	}
	return ck, nil
}

func splitCompositeKey(compositeKey string) (string, []string, error) {
	componentIndex := 1
	var components []string
	for i := 1; i < len(compositeKey); i++ {
		if compositeKey[i] == minUnicodeRuneValue {
			components = append(components, compositeKey[componentIndex:i])
			componentIndex = i + 1
		}
	}
	return components[0], components[1:], nil
}

func validateCompositeKeyAttribute(str string) error {
	if !utf8.ValidString(str) {
		return errors.Errorf("not a valid utf8 string: [%x]", str)
	}
	for index, runeValue := range str {
		if runeValue == minUnicodeRuneValue || runeValue == maxUnicodeRuneValue {
			return errors.Errorf(`input contain unicode %#U starting at position [%d]. %#U and %#U are not allowed in the input attribute of a composite key`,
				runeValue, index, minUnicodeRuneValue, maxUnicodeRuneValue)
		}
	}
	return nil
}
