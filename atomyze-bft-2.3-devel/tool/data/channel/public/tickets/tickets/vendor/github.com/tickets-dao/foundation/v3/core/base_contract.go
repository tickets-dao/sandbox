package core

import (
	"encoding/hex"
	"log"
	"sort"
	"strconv"

	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
	pb "github.com/tickets-dao/foundation/v3/proto"
)

type BaseContract struct {
	id           string
	stub         shim.ChaincodeStubInterface
	methods      []string
	allowedMspID string
	atomyzeSKI   []byte
	initArgs     []string
	noncePrefix  StateKey
}

func (bc *BaseContract) baseContractInit(cc BaseContractInterface) {
	bc.id = cc.GetID()
}

func (bc *BaseContract) GetStub() shim.ChaincodeStubInterface {
	return bc.stub
}

func (bc *BaseContract) GetCreatorSKI() string {
	stub, ok := bc.stub.(*batchTxStub)
	if ok {
		return stub.creatorSKI
	}
	log.Println("Couldn't get creatorSKI because stub is not batchTxStub")
	return ""
}

func (bc *BaseContract) GetMethods() []string {
	return bc.methods
}

func (bc *BaseContract) addMethod(mm string) {
	bc.methods = append(bc.methods, mm)
	sort.Strings(bc.methods)
}

func (bc *BaseContract) setStubAndInitArgs(
	stub shim.ChaincodeStubInterface,
	allowedMspID string,
	atomyzeSKI []byte,
	args []string,
	noncePrefix StateKey,
) {
	bc.stub = stub
	bc.allowedMspID = allowedMspID
	bc.atomyzeSKI = atomyzeSKI
	bc.initArgs = args
	bc.noncePrefix = noncePrefix
}

func (bc *BaseContract) GetAllowedMspID() string {
	return bc.allowedMspID
}

func (bc *BaseContract) GetAtomyzeSKI() []byte {
	return bc.atomyzeSKI
}

func (bc *BaseContract) GetInitArg(idx int) string {
	return bc.initArgs[idx]
}

func (bc *BaseContract) GetInitArgsLen() int {
	return len(bc.initArgs)
}

func (bc *BaseContract) QueryGetNonce(owner *types.Address) (string, error) {
	prefix := hex.EncodeToString([]byte{byte(bc.noncePrefix)})
	key, err := bc.stub.CreateCompositeKey(prefix, []string{owner.String()})
	if err != nil {
		return "", err
	}

	data, err := bc.stub.GetState(key)
	if err != nil {
		return "", err
	}

	exist := new(big.Int).String()

	lastNonce := new(pb.Nonce)
	if len(data) > 0 {
		if err = proto.Unmarshal(data, lastNonce); err != nil {
			// предположим, что это старый нонс
			lastNonce.Nonce = []uint64{new(big.Int).SetBytes(data).Uint64()}
		}
		exist = strconv.FormatUint(lastNonce.Nonce[len(lastNonce.Nonce)-1], 10) //nolint:gomnd
	}

	return exist, nil
}

type BaseContractInterface interface { //nolint:interfacebloat
	GetStub() shim.ChaincodeStubInterface
	addMethod(string)
	setStubAndInitArgs(shim.ChaincodeStubInterface, string, []byte, []string, StateKey)
	GetID() string
	baseContractInit(BaseContractInterface)

	TokenBalanceTransfer(from *types.Address, to *types.Address, amount *big.Int, reason string) error
	AllowedBalanceTransfer(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error

	TokenBalanceGet(address *types.Address) (*big.Int, error)
	TokenBalanceAdd(address *types.Address, amount *big.Int, reason string) error
	TokenBalanceSub(address *types.Address, amount *big.Int, reason string) error

	AllowedBalanceGet(token string, address *types.Address) (*big.Int, error)
	AllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error
	AllowedBalanceSub(token string, address *types.Address, amount *big.Int, reason string) error

	AllowedBalanceGetAll(address *types.Address) (map[string]string, error)

	tokenBalanceAdd(address *types.Address, amount *big.Int, token string) error

	IndustrialBalanceGet(address *types.Address) (map[string]string, error)
	IndustrialBalanceTransfer(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error
	IndustrialBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error
	IndustrialBalanceSub(token string, address *types.Address, amount *big.Int, reason string) error

	AllowedIndustrialBalanceAdd(address *types.Address, industrialAssets []*pb.Asset, reason string) error
	AllowedIndustrialBalanceSub(address *types.Address, industrialAssets []*pb.Asset, reason string) error
	AllowedIndustrialBalanceTransfer(from *types.Address, to *types.Address, industrialAssets []*pb.Asset, reason string) error
}
