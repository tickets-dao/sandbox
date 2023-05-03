package main

import (
	"log"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/tickets-dao/chaincode/logging"
	"github.com/tickets-dao/chaincode/token"
	"github.com/tickets-dao/foundation/v3/core"
)

func main() {
	c := token.NewContract()

	lg := logging.NewHTTPLogger("main")

	lg.Warning("starting tickets, issuer: '%s'", c.Issuer())

	cc, err := core.NewChainCode(c, "org0", nil)
	if err != nil {
		log.Fatal(err)
	}

	lg.Warning("created new chaincode tickets, going to start it\n")

	lg2 := logging.NewHTTPLogger("main.shutdown")

	lg2.Debug("all done")

	if err := shim.Start(cc); err != nil {
		log.Fatal(err)
	}
}
