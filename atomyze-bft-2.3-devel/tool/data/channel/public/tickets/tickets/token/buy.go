package token

import (
	"encoding/json"
	"fmt"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
	"strconv"
	"strings"
	"time"
)

const rubCurrency = "RUB"

func (con *Contract) TxBuy(sender *types.Sender, eventID, categoryName, nowString string, row, number int) (TransferEvent, error) {
	lg.Infof("TxBuy start event id: '%s'", eventID)
	now, err := time.Parse(timeLayout, nowString)
	if err != nil {
		return TransferEvent{}, fmt.Errorf("failed to parse now time from '%s': %v", nowString, err)
	}

	issuer, _, err := parseEventID(eventID)
	if err != nil {
		return TransferEvent{}, err
	}

	event, err := con.getEventByID(eventID)
	if err != nil {
		return TransferEvent{}, fmt.Errorf("failed to get event by id: %v", err)
	}

	if now.After(event.StartTime.Add(timeThreshold)) {
		//концерт уже начался, нельзя погасить
		return TransferEvent{}, fmt.Errorf("now '%s' is after concert start '%s', couldn't burn ticket", now, event.StartTime)
	}

	balances, err := con.IndustrialBalanceGet(issuer)
	if err != nil {
		return TransferEvent{}, fmt.Errorf("failed to get industrial balances of sender '%s': %v", sender.Address(), err)
	}

	lg.Infof("got issuer balances: %v", balances)

	ticketKey := createTicketID(eventID, categoryName, row, number)
	lg.Infof("buying ticket '%s'", ticketKey)

	ticketIndustrial, ok := balances[ticketKey]
	if !ok {
		return TransferEvent{}, fmt.Errorf("unathorized for ticket '%s'", ticketKey)
	}

	lg.Infof("got industrial ticket: %s", ticketIndustrial)

	pricesMap, err := con.getPricesMap(eventID)
	if err != nil {
		return TransferEvent{}, err
	}

	price, ok := pricesMap[categoryName]
	if !ok {
		return TransferEvent{}, fmt.Errorf("unknown category '%s' in map '%v'", categoryName, pricesMap)
	}

	if price == nil {
		lg.Errorf("got nil price for category '%s' from map %v", categoryName, pricesMap)
	}

	err = con.AllowedBalanceTransfer(rubCurrency, sender.Address(), issuer, price, "ticket buy")
	if err != nil {
		return TransferEvent{}, fmt.Errorf("failed to transfer fiat to issuer: %v", err)
	}

	err = con.IndustrialBalanceTransfer(ticketKey, issuer, sender.Address(), new(big.Int).SetInt64(1), "ticket buy")
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
		From:   issuer.String(),
		To:     sender.Address().String(),
		Price:  price.Int64(),
		Ticker: ticketKey,
	}

	lg.Infof("all done with event '%+v'", transferEvent)

	return transferEvent, nil
}

func (con *Contract) NBTxAddAllowedBalance(sender *types.Sender) error {
	return con.AllowedBalanceAdd(rubCurrency, sender.Address(), big.NewInt(2000), "test increase")
}

func parseEventID(eventID string) (*types.Address, int, error) {
	eventIDParts := strings.Split(eventID, "::")
	if len(eventIDParts) != 2 {
		return nil, 0, fmt.Errorf("expected event id '%s' be in format '<issuer_address>::<integer event number>'", eventID)
	}

	address, err := types.AddrFromBase58Check(eventIDParts[0])
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get address from '%s': %v", eventIDParts[0], err)
	}

	eventNum, err := strconv.ParseInt(eventIDParts[1], 10, 32)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse event num from '%s': %v", eventIDParts[1], err)
	}

	return address, int(eventNum), nil
}
