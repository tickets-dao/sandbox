package core

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
	pb "github.com/tickets-dao/foundation/v3/proto"
)

type StateKey byte

// StateKey constants
const (
	StateKeyNonce StateKey = iota + 42 // Этот префикс используется для нонсов у RU и CH
	StateKeyTokenBalance
	StateKeyAllowedBalance
	StateKeyGivenBalance
	StateKeyLockedTokenBalance
	StateKeyLockedAllowedBalance
	StateKeyPassedNonce // Этот префикс используется для нонсов у US
)

func balanceGet(stub shim.ChaincodeStubInterface, tokenType StateKey, addr *types.Address, path ...string) (string, *big.Int, error) {
	prefix := hex.EncodeToString([]byte{byte(tokenType)})
	key, err := stub.CreateCompositeKey(prefix, append([]string{addr.String()}, path...))
	if err != nil {
		return key, nil, err
	}
	data, err := stub.GetState(key)
	if err != nil {
		return key, nil, err
	}
	return key, new(big.Int).SetBytes(data), nil
}

func balanceSub(stub shim.ChaincodeStubInterface, tokenType StateKey, addr *types.Address, amount *big.Int, path ...string) error {
	if amount.Cmp(big.NewInt(0)) < 0 {
		return errors.New("amount should be positive")
	}
	key, balance, err := balanceGet(stub, tokenType, addr, path...)
	if err != nil {
		return err
	}
	if balance.Cmp(amount) < 0 {
		return errors.New("insufficient funds to process")
	}
	return stub.PutState(key, new(big.Int).Sub(balance, amount).Bytes())
}

func balanceAdd(stub shim.ChaincodeStubInterface, tokenType StateKey, addr *types.Address, amount *big.Int, path ...string) error {
	if amount.Cmp(big.NewInt(0)) < 0 {
		return errors.New("amount should be positive")
	}
	key, balance, err := balanceGet(stub, tokenType, addr, path...)
	if err != nil {
		return err
	}
	return stub.PutState(key, new(big.Int).Add(balance, amount).Bytes())
}

func balanceTransfer(stub shim.ChaincodeStubInterface, tokenType StateKey, from *types.Address, to *types.Address, amount *big.Int, path ...string) error {
	if err := balanceSub(stub, tokenType, from, amount, path...); err != nil {
		return err
	}
	return balanceAdd(stub, tokenType, to, amount, path...)
}

func (bc *BaseContract) tokenBalanceSub(address *types.Address, amount *big.Int, token string) error {
	parts := strings.Split(token, "_")
	if len(parts) > 1 {
		return balanceSub(bc.stub, StateKeyTokenBalance, address, amount, parts[len(parts)-1])
	}
	return balanceSub(bc.stub, StateKeyTokenBalance, address, amount)
}

func (bc *BaseContract) tokenBalanceAdd(address *types.Address, amount *big.Int, token string) error {
	parts := strings.Split(token, "_")
	if len(parts) > 1 {
		return balanceAdd(bc.stub, StateKeyTokenBalance, address, amount, parts[len(parts)-1])
	}
	return balanceAdd(bc.stub, StateKeyTokenBalance, address, amount)
}

func balanceList(stub shim.ChaincodeStubInterface, tokenType StateKey, address *types.Address) (map[string]string, error) {
	prefix := hex.EncodeToString([]byte{byte(tokenType)})
	iter, err := stub.GetStateByPartialCompositeKey(prefix, []string{address.String()})
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = iter.Close()
	}()

	res := make(map[string]string)
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return nil, err
		}
		_, keyParts, err := stub.SplitCompositeKey(kv.Key)
		if err != nil {
			return nil, err
		}
		if len(keyParts) < 2 { //nolint:gomnd
			return nil, fmt.Errorf("incorrect composite key %s (two-part key expected)", kv.Key)
		}
		res[keyParts[1]] = new(big.Int).SetBytes(kv.Value).String()
	}
	return res, nil
}

func (bc *BaseContract) IndustrialBalanceGet(address *types.Address) (map[string]string, error) {
	return balanceList(bc.stub, StateKeyTokenBalance, address)
}

func (bc *BaseContract) IndustrialBalanceTransfer(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if stub, ok := bc.GetStub().(*batchTxStub); ok {
		stub.AddAccountingRecord(bc.id+"_"+token, from, to, amount, reason)
	}
	return balanceTransfer(bc.stub, StateKeyTokenBalance, from, to, amount, token)
}

func (bc *BaseContract) IndustrialBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if stub, ok := bc.GetStub().(*batchTxStub); ok {
		stub.AddAccountingRecord(bc.id+"_"+token, &types.Address{}, address, amount, reason)
	}
	return balanceAdd(bc.stub, StateKeyTokenBalance, address, amount, token)
}

func (bc *BaseContract) IndustrialBalanceSub(token string, address *types.Address, amount *big.Int, reason string) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if stub, ok := bc.GetStub().(*batchTxStub); ok {
		stub.AddAccountingRecord(bc.id+"_"+token, address, &types.Address{}, amount, reason)
	}
	return balanceSub(bc.stub, StateKeyTokenBalance, address, amount, token)
}

func (bc *BaseContract) TokenBalanceTransfer(from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	if stub, ok := bc.GetStub().(*batchTxStub); ok {
		stub.AddAccountingRecord(bc.id, from, to, amount, reason)
	}
	return balanceTransfer(bc.stub, StateKeyTokenBalance, from, to, amount)
}

func (bc *BaseContract) AllowedBalanceTransfer(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	if stub, ok := bc.GetStub().(*batchTxStub); ok {
		stub.AddAccountingRecord(token, from, to, amount, reason)
	}
	return balanceTransfer(bc.stub, StateKeyAllowedBalance, from, to, amount, token)
}

func (bc *BaseContract) TokenBalanceGet(address *types.Address) (*big.Int, error) {
	_, balance, err := balanceGet(bc.stub, StateKeyTokenBalance, address)
	return balance, err
}

func (bc *BaseContract) TokenBalanceAdd(address *types.Address, amount *big.Int, reason string) error {
	if stub, ok := bc.GetStub().(*batchTxStub); ok {
		stub.AddAccountingRecord(bc.id, &types.Address{}, address, amount, reason)
	}
	return balanceAdd(bc.stub, StateKeyTokenBalance, address, amount)
}

func (bc *BaseContract) TokenBalanceSub(address *types.Address, amount *big.Int, reason string) error {
	if stub, ok := bc.GetStub().(*batchTxStub); ok {
		stub.AddAccountingRecord(bc.id, address, &types.Address{}, amount, reason)
	}
	return balanceSub(bc.stub, StateKeyTokenBalance, address, amount)
}

func (bc *BaseContract) TokenBalanceGetLocked(address *types.Address) (*big.Int, error) {
	_, balance, err := balanceGet(bc.stub, StateKeyLockedTokenBalance, address)
	return balance, err
}

func (bc *BaseContract) TokenBalanceLock(address *types.Address, amount *big.Int) error {
	if err := balanceSub(bc.stub, StateKeyTokenBalance, address, amount); err != nil {
		return err
	}
	return balanceAdd(bc.stub, StateKeyLockedTokenBalance, address, amount)
}

func (bc *BaseContract) TokenBalanceUnlock(address *types.Address, amount *big.Int) error {
	if err := balanceSub(bc.stub, StateKeyLockedTokenBalance, address, amount); err != nil {
		return err
	}
	return balanceAdd(bc.stub, StateKeyTokenBalance, address, amount)
}

func (bc *BaseContract) TokenBalanceTransferLocked(from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	if stub, ok := bc.GetStub().(*batchTxStub); ok {
		stub.AddAccountingRecord(bc.id, from, to, amount, reason)
	}
	if err := balanceSub(bc.stub, StateKeyLockedTokenBalance, from, amount); err != nil {
		return err
	}
	return balanceAdd(bc.stub, StateKeyTokenBalance, to, amount)
}

func (bc *BaseContract) TokenBalanceBurnLocked(address *types.Address, amount *big.Int, reason string) error {
	if stub, ok := bc.GetStub().(*batchTxStub); ok {
		stub.AddAccountingRecord(bc.id, address, &types.Address{}, amount, reason)
	}
	return balanceSub(bc.stub, StateKeyLockedTokenBalance, address, amount)
}

func (bc *BaseContract) AllowedBalanceGet(token string, address *types.Address) (*big.Int, error) {
	_, balance, err := balanceGet(bc.stub, StateKeyAllowedBalance, address, token)
	return balance, err
}

func (bc *BaseContract) AllowedBalanceAdd(token string, address *types.Address, amount *big.Int, reason string) error {
	if stub, ok := bc.GetStub().(*batchTxStub); ok {
		stub.AddAccountingRecord(token, &types.Address{}, address, amount, reason)
	}
	return balanceAdd(bc.stub, StateKeyAllowedBalance, address, amount, token)
}

func (bc *BaseContract) AllowedBalanceSub(token string, address *types.Address, amount *big.Int, reason string) error {
	if stub, ok := bc.GetStub().(*batchTxStub); ok {
		stub.AddAccountingRecord(token, address, &types.Address{}, amount, reason)
	}
	return balanceSub(bc.stub, StateKeyAllowedBalance, address, amount, token)
}

func (bc *BaseContract) AllowedIndustrialBalanceTransfer(from *types.Address, to *types.Address, industrialAssets []*pb.Asset, reason string) error {
	for _, industrialAsset := range industrialAssets {
		amount := new(big.Int).SetBytes(industrialAsset.Amount)
		if stub, ok := bc.GetStub().(*batchTxStub); ok {
			stub.AddAccountingRecord(industrialAsset.Group, from, to, amount, reason)
		}
		if err := balanceTransfer(bc.stub, StateKeyAllowedBalance, from, to, amount, industrialAsset.Group); err != nil {
			return err
		}
	}

	return nil
}

func (bc *BaseContract) AllowedIndustrialBalanceAdd(address *types.Address, industrialAssets []*pb.Asset, reason string) error {
	for _, industrialAsset := range industrialAssets {
		amount := new(big.Int).SetBytes(industrialAsset.Amount)
		if stub, ok := bc.GetStub().(*batchTxStub); ok {
			stub.AddAccountingRecord(industrialAsset.Group, &types.Address{}, address, amount, reason)
		}
		if err := balanceAdd(bc.stub, StateKeyAllowedBalance, address, amount, industrialAsset.Group); err != nil {
			return err
		}
	}

	return nil
}

func (bc *BaseContract) AllowedIndustrialBalanceSub(address *types.Address, industrialAssets []*pb.Asset, reason string) error {
	for _, asset := range industrialAssets {
		amount := new(big.Int).SetBytes(asset.Amount)
		if stub, ok := bc.GetStub().(*batchTxStub); ok {
			stub.AddAccountingRecord(asset.Group, address, &types.Address{}, amount, reason)
		}
		if err := balanceSub(bc.stub, StateKeyAllowedBalance, address, amount, asset.Group); err != nil {
			return err
		}
	}

	return nil
}

func (bc *BaseContract) AllowedBalanceLock(token string, address *types.Address, amount *big.Int) error {
	if err := balanceSub(bc.stub, StateKeyAllowedBalance, address, amount, token); err != nil {
		return err
	}
	return balanceAdd(bc.stub, StateKeyLockedAllowedBalance, address, amount, token)
}

func (bc *BaseContract) AllowedBalanceUnLock(token string, address *types.Address, amount *big.Int) error {
	if err := balanceSub(bc.stub, StateKeyLockedAllowedBalance, address, amount, token); err != nil {
		return err
	}
	return balanceAdd(bc.stub, StateKeyAllowedBalance, address, amount, token)
}

func (bc *BaseContract) AllowedBalanceTransferLocked(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	if stub, ok := bc.GetStub().(*batchTxStub); ok {
		stub.AddAccountingRecord(token, from, to, amount, reason)
	}
	if err := balanceSub(bc.stub, StateKeyLockedAllowedBalance, from, amount, token); err != nil {
		return err
	}
	return balanceAdd(bc.stub, StateKeyAllowedBalance, to, amount, token)
}

func (bc *BaseContract) AllowedBalanceBurnLocked(token string, address *types.Address, amount *big.Int, reason string) error {
	if stub, ok := bc.GetStub().(*batchTxStub); ok {
		stub.AddAccountingRecord(token, address, &types.Address{}, amount, reason)
	}
	return balanceSub(bc.stub, StateKeyLockedAllowedBalance, address, amount, token)
}

func (bc *BaseContract) IndustrialBalanceGetLocked(address *types.Address) (map[string]string, error) {
	return balanceList(bc.stub, StateKeyLockedTokenBalance, address)
}

func (bc *BaseContract) IndustrialBalanceLock(token string, address *types.Address, amount *big.Int) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if err := balanceSub(bc.stub, StateKeyTokenBalance, address, amount, token); err != nil {
		return err
	}
	return balanceAdd(bc.stub, StateKeyLockedTokenBalance, address, amount, token)
}

func (bc *BaseContract) IndustrialBalanceUnLock(token string, address *types.Address, amount *big.Int) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if err := balanceSub(bc.stub, StateKeyLockedTokenBalance, address, amount, token); err != nil {
		return err
	}
	return balanceAdd(bc.stub, StateKeyTokenBalance, address, amount, token)
}

func (bc *BaseContract) IndustrialBalanceTransferLocked(token string, from *types.Address, to *types.Address, amount *big.Int, reason string) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if stub, ok := bc.GetStub().(*batchTxStub); ok {
		stub.AddAccountingRecord(bc.id+"_"+token, from, to, amount, reason)
	}

	if err := balanceSub(bc.stub, StateKeyLockedTokenBalance, from, amount, token); err != nil {
		return err
	}
	return balanceAdd(bc.stub, StateKeyTokenBalance, to, amount, token)
}

func (bc *BaseContract) IndustrialBalanceBurnLocked(token string, address *types.Address, amount *big.Int, reason string) error {
	parts := strings.Split(token, "_")
	token = parts[len(parts)-1]
	if stub, ok := bc.GetStub().(*batchTxStub); ok {
		stub.AddAccountingRecord(bc.id+"_"+token, address, &types.Address{}, amount, reason)
	}
	return balanceSub(bc.stub, StateKeyLockedTokenBalance, address, amount, token)
}

func (bc *BaseContract) AllowedBalanceGetAll(addr *types.Address) (map[string]string, error) {
	return balanceList(bc.stub, StateKeyAllowedBalance, addr)
}

func givenBalanceGet(stub shim.ChaincodeStubInterface, contract string) (string, *big.Int, error) {
	prefix := hex.EncodeToString([]byte{byte(StateKeyGivenBalance)})
	key, err := stub.CreateCompositeKey(prefix, []string{contract})
	if err != nil {
		return key, nil, err
	}
	data, err := stub.GetState(key)
	if err != nil {
		return key, nil, err
	}
	return key, new(big.Int).SetBytes(data), nil
}

func GivenBalanceAdd(stub shim.ChaincodeStubInterface, contract string, amount *big.Int) error {
	key, balance, err := givenBalanceGet(stub, contract)
	if err != nil {
		return err
	}
	return stub.PutState(key, new(big.Int).Add(balance, amount).Bytes())
}

func GivenBalanceSub(stub shim.ChaincodeStubInterface, contract string, amount *big.Int) error {
	key, balance, err := givenBalanceGet(stub, contract)
	if err != nil {
		return err
	}
	if balance.Cmp(amount) < 0 {
		return errors.New("insufficient funds to process")
	}
	return stub.PutState(key, new(big.Int).Sub(balance, amount).Bytes())
}
