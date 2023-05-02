package mock

import (
	"fmt"

	"github.com/hyperledger/fabric-chaincode-go/shim"
)

type Right struct {
	Channel   string
	Chaincode string
	Role      string
	Operation string
	Address   string
}

func (r Right) IsValid() error {
	if len(r.Channel) == 0 {
		return fmt.Errorf("right is broken, channel is not set")
	}

	if len(r.Chaincode) == 0 {
		return fmt.Errorf("right is broken, chaincode is not set")
	}

	if len(r.Role) == 0 {
		return fmt.Errorf("right is broken, role is not set")
	}

	if len(r.Operation) == 0 {
		return fmt.Errorf("right is broken, operation is not set")
	}

	if len(r.Address) == 0 {
		return fmt.Errorf("right is broken, address is not set")
	}

	return nil
}

type operation string

const (
	AddRights    operation = "addRights"
	RemoveRights operation = "removeRights"
)

func (w *Wallet) AddAccountRight(right *Right) error {
	return w.modifyRight(AddRights, right)
}

func (w *Wallet) RemoveAccountRight(right *Right) error {
	return w.modifyRight(RemoveRights, right)
}

func (w *Wallet) modifyRight(opFn operation, right *Right) error {
	if right == nil {
		return fmt.Errorf("right is not set")
	}

	validationErr := right.IsValid()
	if validationErr != nil {
		return validationErr
	}

	params := [][]byte{
		[]byte(opFn),
		[]byte(right.Channel),
		[]byte(right.Chaincode),
		[]byte(right.Role),
		[]byte(right.Operation),
		[]byte(right.Address),
	}
	const acl = "acl"
	aclstub := w.ledger.GetStub(acl)
	aclstub.TxID = txIDGen()
	aclstub.MockPeerChaincodeWithChannel(acl, aclstub, acl)

	rsp := aclstub.InvokeChaincode(acl, params, acl)
	if rsp.Status != shim.OK {
		return fmt.Errorf(rsp.Message)
	}

	return nil
}
