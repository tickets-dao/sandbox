package token

import (
	"encoding/json"
	"fmt"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
)

const rubCurrency = "RUB"

func (con *Contract) NBTxBuy(sender *types.Sender, categoryName string, sector, row, number int) (TransferEvent, error) {
	lg.Infof("TxBuy start, metadata: %+v, issuer: '%s'", contractMetadata, con.Issuer())

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

	pricesMap, err := con.getPricesMap()
	if err != nil {
		return TransferEvent{}, err
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

func (con *Contract) NBTxAddAlowedBalance(sender *types.Sender) error {
	return con.AllowedBalanceAdd(rubCurrency, sender.Address(), big.NewInt(2000), "test increase")
}

func (con Contract) getPricesMap() (map[string]*big.Int, error) {
	pricesMapBytes, err := con.GetStub().GetState(joinStateKey(con.Issuer().String(), pricesMapStateSubKey))
	if err != nil {
		return nil, fmt.Errorf("failed to get prices map: %v", err)
	}

	var pricesMap map[string]*big.Int

	if err = json.Unmarshal(pricesMapBytes, &pricesMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prices map: %v", err)
	}

	return pricesMap, nil
}
