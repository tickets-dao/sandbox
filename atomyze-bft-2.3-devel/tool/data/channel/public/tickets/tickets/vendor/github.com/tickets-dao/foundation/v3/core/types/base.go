package types

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/tickets-dao/foundation/v3/core/types/big"
)

var BaseTypes = map[string]interface{}{
	"string": func(_ string, _ shim.ChaincodeStubInterface, in string) (string, error) {
		return in, nil
	},
	"int": func(_ int, _ shim.ChaincodeStubInterface, in string) (int, error) {
		return strconv.Atoi(in)
	},
	"bool": func(_ bool, _ shim.ChaincodeStubInterface, in string) (bool, error) {
		return in != "" && strings.ToLower(in) != "false", nil
	},
	"int64": func(_ int64, _ shim.ChaincodeStubInterface, in string) (int64, error) {
		return strconv.ParseInt(in, 10, 64) //nolint:gomnd
	},
	"uint32": func(_ uint32, _ shim.ChaincodeStubInterface, in string) (uint32, error) {
		v, err := strconv.ParseUint(in, 10, 32) //nolint:gomnd
		if err != nil {
			return 0, err
		}
		return uint32(v), nil
	},
	"uint64": func(_ uint64, _ shim.ChaincodeStubInterface, in string) (uint64, error) {
		v, err := strconv.ParseUint(in, 10, 64) //nolint:gomnd
		if err != nil {
			return 0, err
		}
		return v, nil
	},
	"float64": func(_ float64, _ shim.ChaincodeStubInterface, in string) (float64, error) {
		return strconv.ParseFloat(in, 64) //nolint:gomnd
	},
	"*big.Int": func(_ *big.Int, _ shim.ChaincodeStubInterface, in string) (*big.Int, error) {
		value, ok := new(big.Int).SetString(in, 10) //nolint:gomnd
		if !ok {
			return nil, fmt.Errorf("couldn't convert %s to bigint", in)
		}
		if value.Cmp(big.NewInt(0)) < 0 {
			return nil, fmt.Errorf("value %s should be positive", in)
		}
		return value, nil
	},
	"[]uint8": func(_ []uint8, stub shim.ChaincodeStubInterface, in string) ([]uint8, error) {
		return base58.Decode(in), nil
	},
}
