package main

import (
	"log"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/tickets-dao/foundation/v3/core"
	"github.com/tickets-dao/foundation/v3/token"
	mintable "github.com/tickets-dao/sandbox/cc/token"
)

func main() {
	cct := mintable.NewMintableToken(token.BaseToken{
		Name:            "Atomyze USD​",
		Symbol:          "CC",
		Decimals:        8,
		UnderlyingAsset: "USD",
	})
	cc, err := core.NewChainCode(cct, "org0", nil)
	if err != nil {
		log.Fatal(err)
	}
	if err := shim.Start(cc); err != nil {
		log.Fatal(err)
	}
}
