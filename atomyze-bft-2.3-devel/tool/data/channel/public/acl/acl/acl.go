package main

import (
	"log"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	acl "gitlab.n-t.io/acl/token"
)

const OrgMSP = "org0"

func main() {
	if err := shim.Start(acl.New(OrgMSP)); err != nil {
		log.Fatal(err)
	}
}
