package token

import (
	"encoding/json"
	"fmt"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
)

type IssuerInfo struct {
	Parent      *types.Address `json:"parent"`
	NextEventID int            `json:"next_event_id"`
}

func (con *Contract) NBTxMarkUserAsIssuer(sender *types.Sender, issuerCandidate *types.Address) (IssuerInfo, error) {
	lg.Infof("starting mark user as issuer: sender '%s', issuer candidate: '%s'", sender.Address(), issuerCandidate)

	senderInfo, err := con.getIssuerInfo(sender.Address())
	if err != nil {
		return IssuerInfo{}, fmt.Errorf("failed to get issuer info for sender: %v", err)
	}

	if senderInfo.Parent == nil {
		return IssuerInfo{}, fmt.Errorf("unathorized: sender must be issuer")
	}

	newIssuerInfo, err := con.getIssuerInfo(issuerCandidate)
	if err != nil {
		return IssuerInfo{}, fmt.Errorf("failed to get issuer info for issuer candidate: %v", err)
	}

	if newIssuerInfo.Parent != nil {
		return IssuerInfo{}, fmt.Errorf("issuer candidate is already issuer: %v", newIssuerInfo)
	}

	newIssuerInfo = IssuerInfo{Parent: sender.Address(), NextEventID: 1}

	newIssuerBytes, _ := json.Marshal(newIssuerInfo)
	if err = con.GetStub().PutState(issuerCandidate.String(), newIssuerBytes); err != nil {
		return IssuerInfo{}, fmt.Errorf("failed to put state for new issuer: %v", err)
	}

	return newIssuerInfo, nil
}

// NBTxInitialize - initializes chaincode
func (con *Contract) NBTxEmission(sender *types.Sender) error {
	lg.Infof("this is nbtx initialize for sender %s, issuer: '%s'\n", sender.Address().String(), con.Issuer())

	lg.Infof("price categories: %+v\n", priceCategories)
	//TODO check sender is issuer

	categoriesMap := make(map[string]*big.Int, len(priceCategories))
	for _, category := range priceCategories {
		if _, ok := categoriesMap[category.Name]; ok {
			return fmt.Errorf("category '%s' is used more than once", category.Name)
		}

		categoriesMap[category.Name] = category.Price
	}

	issuerAddress := sender.Address().String()

	categoriesMapBytes, err := json.Marshal(categoriesMap)
	if err != nil {
		return err
	}

	lg.Infof("categories map: '%s'\n", string(categoriesMapBytes))

	if err = con.GetStub().PutState(joinStateKey(issuerAddress, pricesMapStateSubKey), categoriesMapBytes); err != nil {
		return fmt.Errorf("failed to save categories map state: %v", err)
	}

	if err = con.GetStub().PutState(joinStateKey(issuerAddress, ticketersStateSubKey), []byte("[]")); err != nil {
		return fmt.Errorf("failed to save ticketers addreses: %v", err)
	}

	if err = con.GetStub().PutState(joinStateKey(issuerAddress, issuerBalanceStateSubKey), []byte("0")); err != nil {
		return fmt.Errorf("failed to save ticketers addreses: %v", err)
	}

	if err = con.GetStub().PutState(joinStateKey(issuerAddress, buyBackRateStateSubKey), []byte("1")); err != nil {
		return fmt.Errorf("failed to save ticketers addreses: %v", err)
	}

	// билеты, которые нужно выпустить, соответствует ключу в стейте
	ticketKeys := make([]string, 0, len(priceCategories))

	for _, category := range priceCategories {
		for _, seat := range category.Seats {
			ticketID := con.createTicketID(category.Name, seat.Sector, seat.Row, seat.Number)
			ticketKeys = append(ticketKeys, ticketID)
		}
	}

	for _, ticketID := range ticketKeys {
		lg.Infof("got ticketID '%s'", ticketID)
		err = con.IndustrialBalanceAdd("ticket_"+ticketID, con.Issuer(), new(big.Int).SetInt64(1), "initial emission")
		if err != nil {
			return fmt.Errorf("failed to emit ticket %s: %v", ticketID, err)
		}
	}

	lg.Infof("all done, returning nil error\n")

	return nil
}

func (con *Contract) QueryIssuerInfo(sender *types.Sender) (IssuerInfo, error) {
	lg.Infof("query issuer info for sender: '%s'", sender.Address())

	return con.getIssuerInfo(sender.Address())
}

func (con *Contract) getIssuerInfo(sender *types.Address) (IssuerInfo, error) {
	stub := con.GetStub()

	senderInfoBytes, err := stub.GetState(sender.String())
	if err != nil {
		return IssuerInfo{}, fmt.Errorf("failed to get state for sender: %v", err)
	}

	var senderInfo IssuerInfo
	err = json.Unmarshal(senderInfoBytes, &senderInfo)
	if err != nil {
		return IssuerInfo{}, fmt.Errorf("failed to unmarshal send info from '%s': %v", string(senderInfoBytes), err)
	}

	return senderInfo, nil
}
