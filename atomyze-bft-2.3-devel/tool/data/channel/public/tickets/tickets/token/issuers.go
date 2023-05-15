package token

import (
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/pkg/errors"
	"github.com/tickets-dao/foundation/v3/core/types"
	"sort"
)

const issuersStatePrefix = "issuers"

var errIssuerNotFound = errors.New("there is no issuer with provided address")

type IssuerInfo struct {
	Parent      *types.Address `json:"parent"`
	ID          *types.Address `json:"id"`
	NextEventID int            `json:"next_event_id"`
	Name        string         `json:"name"`
	Ticketers   []string       `json:"ticketers"`
}

func (con *Contract) NBTxMarkUserAsIssuer(sender *types.Sender, issuerCandidate *types.Address, newName string) (IssuerInfo, error) {
	lg.Infof("starting mark user as issuer: sender '%s', issuer candidate: '%s'", sender.Address(), issuerCandidate)

	_, err := con.getIssuerInfo(sender.Address())
	if err != nil {
		if errors.Is(err, errIssuerNotFound) {
			return IssuerInfo{}, fmt.Errorf("unathorized: sender must be issuer")
		}

		return IssuerInfo{}, fmt.Errorf("failed to get issuer info for sender: %v", err)
	}

	newIssuerInfo, err := con.getIssuerInfo(issuerCandidate)
	if err != nil && !errors.Is(err, errIssuerNotFound) {
		return IssuerInfo{}, fmt.Errorf("failed to get issuer info for issuer candidate: %v", err)
	}

	if newIssuerInfo.Parent != nil {
		return IssuerInfo{}, fmt.Errorf("issuer candidate is already issuer: %v", newIssuerInfo)
	}

	newIssuerInfo = IssuerInfo{Parent: sender.Address(), NextEventID: 1, Name: newName, ID: issuerCandidate}

	if err = con.saveIssuerInfo(issuerCandidate, newIssuerInfo); err != nil {
		return IssuerInfo{}, fmt.Errorf("failed to save issuer info for issuer candidate: %v", err)
	}

	return newIssuerInfo, nil
}

func (con *Contract) QueryIssuerInfo(address *types.Address) (IssuerInfo, error) {
	lg.Infof("query issuer info for address: '%s'", address)

	return con.getIssuerInfo(address)
}

func (con *Contract) QueryAllIssuersInfo() ([]IssuerInfo, error) {
	lg.Info("query all issuers info")

	stub := con.GetStub()

	iterator, err := stub.GetStateByPartialCompositeKey(issuersStatePrefix, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to get iterator: %v", err)
	}

	defer func(iterator shim.StateQueryIteratorInterface) {
		err2 := iterator.Close()
		if err2 != nil {
			lg.Errorf("got error %v on closing iterator", err2)
		}
	}(iterator)

	var allIssuersInfo []IssuerInfo
	var totalCount int

	for iterator.HasNext() {
		totalCount++
		info, err := iterator.Next()
		if err != nil {
			lg.Errorf("failed to get next info from iterator: %v", err)
			continue
		}

		var issuerInfo IssuerInfo
		err = json.Unmarshal(info.Value, &issuerInfo)
		if err != nil {
			lg.Errorf("failed to unmarshal send info from '%s': %v", string(info.Value), err)
			continue
		}

		allIssuersInfo = append(allIssuersInfo, issuerInfo)
	}

	sort.Slice(allIssuersInfo, func(i, j int) bool {
		return allIssuersInfo[i].Name <= allIssuersInfo[j].Name
	})

	lg.Infof("got %d issuers info from %d total count", len(allIssuersInfo), totalCount)

	return allIssuersInfo, nil
}

func (con *Contract) getIssuerInfo(address *types.Address) (IssuerInfo, error) {
	stub := con.GetStub()

	issuerKey, err := stub.CreateCompositeKey(issuersStatePrefix, []string{address.String()})
	if err != nil {
		return IssuerInfo{}, fmt.Errorf("failed to create composite key for address: '%s': %v", address, err)
	}

	senderInfoBytes, err := stub.GetState(issuerKey)
	if err != nil {
		return IssuerInfo{}, fmt.Errorf("failed to get state for address: %v", err)
	}

	if senderInfoBytes == nil {
		return IssuerInfo{}, errIssuerNotFound
	}

	var senderInfo IssuerInfo
	err = json.Unmarshal(senderInfoBytes, &senderInfo)
	if err != nil {
		return IssuerInfo{}, fmt.Errorf("failed to unmarshal send info from '%s': %v", string(senderInfoBytes), err)
	}

	return senderInfo, nil
}

func (con *Contract) saveIssuerInfo(address *types.Address, issuerInfo IssuerInfo) error {
	lg.Infof("saving issuer info for sender '%s': %+v", address, issuerInfo)
	issuerInfoBytes, err := json.Marshal(issuerInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal issuer info: %v", err)
	}

	stub := con.GetStub()

	issuerKey, err := stub.CreateCompositeKey(issuersStatePrefix, []string{address.String()})
	if err != nil {
		return fmt.Errorf("failed to create composite key  for address: '%s': %v", address, err)
	}

	// so we could iterate over all issuers
	err = stub.PutState(issuerKey, issuerInfoBytes)
	if err != nil {

		return fmt.Errorf("failed to save issuer info: %v", err)
	}
	return nil
}
