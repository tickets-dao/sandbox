package core

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"reflect"
	"runtime/debug"
	"strings"

	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/proto"
	"golang.org/x/crypto/sha3"
)

const (
	assertInterfaceErrMsg = "assertion interface -> error is failed"
	minChaincodeArgsCount = 2
)

type NonceCheckFn func(shim.ChaincodeStubInterface, *types.Sender, uint64) error

type ChainCode struct {
	contract          BaseContractInterface
	methods           map[string]*Fn
	allowedMspID      string
	checkInvokerBy    cib
	disableSwaps      bool
	init              *proto.InitArgs
	disableMultiSwaps bool
	txTTL             uint
	batchPrefix       string
	nonceTTL          uint
	noncePrefix       StateKey
	nonceCheckFn      NonceCheckFn
}

func NewChainCode(cc BaseContractInterface, allowedMspID string, options *ContractOptions) (*ChainCode, error) {
	cc.baseContractInit(cc)
	methods, err := ParseContract(cc, options)
	if err != nil {
		return &ChainCode{}, err
	}

	out := &ChainCode{
		contract:     cc,
		allowedMspID: allowedMspID,
		methods:      methods,
		batchPrefix:  batchKey,
		noncePrefix:  StateKeyNonce,
		nonceCheckFn: checkNonce(0, StateKeyNonce),
	}

	if options != nil {
		out.checkInvokerBy = options.CheckInvokerBy
		out.disableSwaps = options.DisableSwaps
		out.disableMultiSwaps = options.DisableMultiSwaps
		out.txTTL = options.TxTTL
		if options.BatchPrefix != "" {
			out.batchPrefix = options.BatchPrefix
		}
		if options.NonceTTL != 0 {
			out.nonceTTL = options.NonceTTL
		}
		if options.IsOtherNoncePrefix {
			out.noncePrefix = StateKeyPassedNonce
		}

		out.nonceCheckFn = checkNonce(out.nonceTTL, out.noncePrefix)
	}

	return out, nil
}

func (cc *ChainCode) Init(stub shim.ChaincodeStubInterface) peer.Response {
	creator, err := stub.GetCreator()
	if err != nil {
		return shim.Error(err.Error())
	}
	var identity msp.SerializedIdentity
	if err = pb.Unmarshal(creator, &identity); err != nil {
		return shim.Error(err.Error())
	}
	if identity.Mspid != cc.allowedMspID {
		return shim.Error("incorrect MSP Id")
	}
	b, _ := pem.Decode(identity.IdBytes)
	parsed, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	ouIsOk := false
	for _, ou := range parsed.Subject.OrganizationalUnit {
		if strings.ToLower(ou) == "admin" {
			ouIsOk = true
		}
	}
	if !ouIsOk {
		return shim.Error("incorrect sender's OU")
	}
	args := stub.GetStringArgs()
	if len(args) < minChaincodeArgsCount {
		return shim.Error("should set ski of atomyze and robot certs")
	}
	atomyzeSKI, err := hex.DecodeString(args[0])
	if err != nil {
		return shim.Error(err.Error())
	}
	robotSKI, err := hex.DecodeString(args[1])
	if err != nil {
		return shim.Error(err.Error())
	}
	data, err := pb.Marshal(&proto.InitArgs{
		AtomyzeSKI: atomyzeSKI,
		RobotSKI:   robotSKI,
		Args:       args[2:],
	})
	if err != nil {
		return shim.Error(err.Error())
	}
	if err = stub.PutState("__init", data); err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}

func (cc *ChainCode) Invoke(stub shim.ChaincodeStubInterface) (r peer.Response) { //nolint:gocognit,funlen
	r = shim.Error("panic invoke")
	defer func() {
		if rc := recover(); rc != nil {
			log.Println("panic invoke\n" + string(debug.Stack()))
		}
	}()

	if cc.init == nil {
		data, err := stub.GetState("__init")
		if err != nil {
			return shim.Error(err.Error())
		}
		var args proto.InitArgs
		if err = pb.Unmarshal(data, &args); err != nil {
			return shim.Error(err.Error())
		}
		cc.init = &args
	}

	_, err := hex.DecodeString(stub.GetTxID())
	if err != nil {
		return shim.Error(fmt.Sprintf("incorrect tx id %s", err.Error()))
	}

	creator, err := stub.GetCreator()
	if err != nil {
		return shim.Error(err.Error())
	}
	var identity msp.SerializedIdentity
	if err = pb.Unmarshal(creator, &identity); err != nil {
		return shim.Error(err.Error())
	}
	b, _ := pem.Decode(identity.IdBytes)
	parsed, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	pk, ok := parsed.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		shim.Error("public key type assertion failed")
	}
	creatorSKI := sha256.Sum256(elliptic.Marshal(pk.Curve, pk.X, pk.Y))

	f, args := stub.GetFunctionAndParameters()
	switch f {
	case "batchExecute":
		hashedCert := sha3.Sum256(creator)
		if !bytes.Equal(hashedCert[:], cc.init.RobotSKI) &&
			!bytes.Equal(creatorSKI[:], cc.init.RobotSKI) {
			return shim.Error("unauthorized")
		}
		return cc.batchExecute(stub, hex.EncodeToString(creatorSKI[:]), args[0])
	case "swapDone":
		if cc.disableSwaps {
			return shim.Error("swaps disabled")
		}
		_, contract := copyContract(cc.contract, stub, cc.allowedMspID, cc.init.AtomyzeSKI, cc.init.Args, cc.noncePrefix)
		return swapUserDone(contract, args[0], args[1])
	case "multiSwapDone":
		if cc.disableMultiSwaps {
			return shim.Error("industrial swaps disabled")
		}
		_, contract := copyContract(cc.contract, stub, cc.allowedMspID, cc.init.AtomyzeSKI, cc.init.Args, cc.noncePrefix)
		return multiSwapUserDone(contract, args[0], args[1])
	}
	method, exists := cc.methods[f]
	if !exists {
		return shim.Error("unknown method")
	}
	if !method.query {
		switch cc.checkInvokerBy {
		case CheckInvokerByMSP:
			if identity.Mspid != cc.allowedMspID {
				return shim.Error("your mspId isn't allowed to invoke")
			}
		case CheckInvokerBySKI:
			if !bytes.Equal(creatorSKI[:], cc.init.AtomyzeSKI) {
				return shim.Error("only specified certificate can invoke")
			}
		}
	}
	if method.noBatch {
		sender, args, _, err := cc.checkAuthIfNeeds(stub, method, f, args, true)
		if err != nil {
			return shim.Error(err.Error())
		}
		args, err = doPrepareToSave(stub, method, args)
		if err != nil {
			return shim.Error(err.Error())
		}
		resp, err := cc.callMethod(stub, method, sender, args)
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(resp)
	}

	sender, args, nonce, err := cc.checkAuthIfNeeds(stub, method, f, args, true)
	if err != nil {
		return shim.Error(err.Error())
	}
	args, err = doPrepareToSave(stub, method, args)
	if err != nil {
		return shim.Error(err.Error())
	}
	if err = cc.saveToBatch(stub, f, creatorSKI[:], sender, args[:len(method.in)], nonce); err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}

func (cc *ChainCode) callMethod(
	stub shim.ChaincodeStubInterface,
	method *Fn,
	sender *proto.Address,
	args []string,
) ([]byte, error) {
	values, err := doConvertToCall(stub, method, args)
	if err != nil {
		return nil, err
	}
	if sender != nil {
		values = append([]reflect.Value{
			reflect.ValueOf(types.NewSenderFromAddr((*types.Address)(sender))),
		}, values...)
	}

	contract, _ := copyContract(cc.contract, stub, cc.allowedMspID, cc.init.AtomyzeSKI, cc.init.Args, cc.noncePrefix)

	out := method.fn.Call(append([]reflect.Value{contract}, values...))
	errInt := out[0].Interface()
	if method.out {
		errInt = out[1].Interface()
	}
	if errInt != nil {
		err, ok := errInt.(error)
		if !ok {
			return nil, errors.New(assertInterfaceErrMsg)
		}
		return nil, err
	}

	if method.out {
		return json.Marshal(out[0].Interface())
	}
	return nil, nil
}

func doConvertToCall(stub shim.ChaincodeStubInterface, method *Fn, args []string) ([]reflect.Value, error) {
	if len(args) < len(method.in) {
		return nil, errors.New("incorrect number of arguments")
	}
	// todo check is args enough
	vArgs := make([]reflect.Value, len(method.in))
	for i := range method.in {
		var impl reflect.Value
		if method.in[i].kind.Kind().String() == "ptr" {
			impl = reflect.New(method.in[i].kind.Elem())
		} else {
			impl = reflect.New(method.in[i].kind).Elem()
		}

		res := method.in[i].convertToCall.Call([]reflect.Value{
			impl,
			reflect.ValueOf(stub), reflect.ValueOf(args[i]),
		})

		if res[1].Interface() != nil {
			err, ok := res[1].Interface().(error)
			if !ok {
				return nil, errors.New(assertInterfaceErrMsg)
			}
			return nil, err
		}
		vArgs[i] = res[0]
	}
	return vArgs, nil
}

func doPrepareToSave(stub shim.ChaincodeStubInterface, method *Fn, args []string) ([]string, error) {
	if len(args) < len(method.in) {
		return nil, errors.New("incorrect number of arguments")
	}
	as := make([]string, len(method.in))
	for i := range method.in {
		var impl reflect.Value
		if method.in[i].kind.Kind().String() == "ptr" {
			impl = reflect.New(method.in[i].kind.Elem())
		} else {
			impl = reflect.New(method.in[i].kind).Elem()
		}

		var ok bool
		if method.in[i].prepareToSave.IsValid() {
			res := method.in[i].prepareToSave.Call([]reflect.Value{
				impl,
				reflect.ValueOf(stub), reflect.ValueOf(args[i]),
			})
			if res[1].Interface() != nil {
				err, ok := res[1].Interface().(error)
				if !ok {
					return nil, errors.New(assertInterfaceErrMsg)
				}
				return nil, err
			}
			as[i], ok = res[0].Interface().(string)
			if !ok {
				return nil, errors.New(assertInterfaceErrMsg)
			}
			continue
		}

		// if method PrepareToSave don't have exists
		// use ConvertToCall to check converting
		res := method.in[i].convertToCall.Call([]reflect.Value{
			impl,
			reflect.ValueOf(stub), reflect.ValueOf(args[i]),
		})
		if res[1].Interface() != nil {
			err, ok := res[1].Interface().(error)
			if !ok {
				return nil, errors.New(assertInterfaceErrMsg)
			}
			return nil, err
		}

		as[i] = args[i] // in this case we don't convert argument
	}
	return as, nil
}

func copyContract(
	orig BaseContractInterface,
	stub shim.ChaincodeStubInterface,
	allowedMspID string,
	atomyzeSKI []byte,
	initArgs []string,
	noncePrefix StateKey,
) (reflect.Value, BaseContractInterface) {
	cp := reflect.New(reflect.ValueOf(orig).Elem().Type())
	val := reflect.ValueOf(orig).Elem()
	for i := 0; i < val.NumField(); i++ {
		if cp.Elem().Field(i).CanSet() {
			cp.Elem().Field(i).Set(val.Field(i))
		}
	}
	contract, ok := cp.Interface().(BaseContractInterface)
	if !ok {
		return cp, nil
	}
	contract.setStubAndInitArgs(stub, allowedMspID, atomyzeSKI, initArgs, noncePrefix)
	return cp, contract
}
