package token

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/tickets-dao/foundation/v3/core"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
)

func joinStateKey(keyParts ...string) string {
	buf := bytes.Buffer{}
	for i := range keyParts {
		buf.WriteString(keyParts[i])
		if i < len(keyParts)-1 {
			buf.WriteString("::")
		}
	}

	return buf.String()
}

func balanceList(stub shim.ChaincodeStubInterface, tokenType core.StateKey, address *types.Address) (map[string]string, error) {
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

func IndustrialBalanceGet(stub shim.ChaincodeStubInterface, address *types.Address) map[string]string {
	list, err := balanceList(stub, core.StateKeyTokenBalance, address)
	if err != nil {
		panic(err)
	}

	return list
}

func AllowedBalanceGetAll(stub shim.ChaincodeStubInterface, addr *types.Address) map[string]string {
	list, err := balanceList(stub, core.StateKeyAllowedBalance, addr)
	if err != nil {
		panic(err)
	}

	return list
}
