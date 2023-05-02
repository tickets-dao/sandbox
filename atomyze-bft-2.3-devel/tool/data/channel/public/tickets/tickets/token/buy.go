package token

import (
	"encoding/json"
	"fmt"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
)

const rubCurrency = "RUB"

func (con *Contract) TxBuy(sender *types.Sender, categoryName string, sector, row, number int) (TransferEvent, error) {
	lg.Infof("TxBuy start")
	issuerAddress := con.Issuer().String()

	ticketKey := con.createTicketID(categoryName, sector, row, number)
	lg.Infof("starting buying ticket '%s'", ticketKey)

	balances, err := con.IndustrialBalanceGet(con.Issuer())
	if err != nil {
		return TransferEvent{}, fmt.Errorf("failed to get industrial balances of sender '%s': %v", sender.Address(), err)
	}

	lg.Infof("got issuer balances: %v", balances)

	ticketIndustrial, ok := balances[ticketKey]
	if !ok {
		return TransferEvent{}, fmt.Errorf("unathorized for ticket '%s'", ticketKey)
	}

	lg.Infof("got industrial ticket: %s", ticketIndustrial)

	pricesMapBytes, err := con.GetStub().GetState(joinStateKey(issuerAddress, pricesMapStateSubKey))
	if err != nil {
		return TransferEvent{}, fmt.Errorf("failed to get prices map: %v", err)
	}

	var pricesMap map[string]*big.Int

	if err = json.Unmarshal(pricesMapBytes, &pricesMap); err != nil {
		return TransferEvent{}, fmt.Errorf("failed to unmarshal prices map: %v", err)
	}

	price, ok := pricesMap[categoryName]
	if !ok {
		return TransferEvent{}, fmt.Errorf("unknown category '%s' in map '%v'", categoryName, pricesMap)
	}

	err = con.AllowedBalanceTransfer(rubCurrency, sender.Address(), con.Issuer(), price, "ticket buy")
	if err != nil {
		return TransferEvent{}, fmt.Errorf("failed to transfer fiat to issuer: %v", err)
	}

	err = con.IndustrialBalanceTransfer(ticketKey, con.Issuer(), sender.Address(), new(big.Int).SetInt64(1), "ticket buy")
	if err != nil {
		return TransferEvent{}, fmt.Errorf("failed to transfer ticket to sender: %v", err)
	}

	ticket := Ticket{LastBuyPrice: price, Owner: sender.Address().String()}

	ticketBytes, err := json.Marshal(ticket)
	if err != nil {
		return TransferEvent{}, fmt.Errorf("faied to marshal ticket: %v", err)
	}

	err = con.GetStub().PutState(ticketKey, ticketBytes)
	if err != nil {
		return TransferEvent{}, fmt.Errorf("failed to update ticket state: %v", err)
	}

	transferEvent := TransferEvent{
		From:   con.Issuer().String(),
		To:     sender.Address().String(),
		Price:  price.Int64(),
		Ticker: ticketKey,
	}

	lg.Infof("all done with event '%+v'", transferEvent)

	return transferEvent, nil
}

func (con *Contract) TxAddAlowedBalance(sender *types.Sender) error {
	return con.AllowedBalanceAdd(rubCurrency, sender.Address(), big.NewInt(2000), "test increase")
}
