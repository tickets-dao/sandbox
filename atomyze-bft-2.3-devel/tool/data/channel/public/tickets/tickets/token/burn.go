package token

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
)

func (con *Contract) NBTxBurn(sender *types.Sender, eventID, categoryName string, row, number int, burningPrivateKey string) (BurnEvent, error) {
	issuer, _, err := parseEventID(eventID)
	if err != nil {
		return BurnEvent{}, err
	}

	if err = con.checkTicketer(issuer, sender.Address()); err != nil {
		return BurnEvent{}, err
	}

	ticketKey := createTicketID(eventID, categoryName, row, number)
	ticket, err := con.getTicket(ticketKey)
	if err != nil {
		return BurnEvent{}, fmt.Errorf("failed to get ticket at '%s': %v", ticketKey, err)
	}

	owner, err := types.AddrFromBase58Check(ticket.Owner)
	if err != nil {
		return BurnEvent{}, fmt.Errorf("failed to parse owner from '%s': %v", ticket.Owner, err)
	}

	burningHash := md5.Sum([]byte(burningPrivateKey))

	if ticket.BurningHash != hex.EncodeToString(burningHash[:]) {
		lg.Warningf(
			"got bad burning key '%s' from '%s', expected '%s'",
			string(burningHash[:]),
			burningPrivateKey,
			ticket.BurningHash,
		)

		return BurnEvent{}, fmt.Errorf("bad burning key, got hash '%s' instead of '%s'", string(burningHash[:]), ticket.BurningHash)
	}

	balances, err := con.IndustrialBalanceGet(owner)
	if err != nil {
		return BurnEvent{}, err
	}

	ticketBalance, ok := balances[ticketKey]
	lg.Infof("balances has ticket: %s, %t", ticketBalance, ok)

	if err = con.IndustrialBalanceLock(ticketKey, owner, big.NewInt(1)); err != nil {
		return BurnEvent{}, fmt.Errorf("failed to lock ticket: %v", err)
	}

	if err = con.IndustrialBalanceBurnLocked(ticketKey, owner, big.NewInt(1), "scanned qr"); err != nil {
		return BurnEvent{}, fmt.Errorf("failed to lock ticket: %v", err)
	}

	return BurnEvent{
		Owner:       ticket.Owner,
		Ticketer:    sender.Address().String(),
		Ticker:      ticketKey,
		BurningHash: ticket.BurningHash,
	}, nil
}

func (con *Contract) getTicket(ticketKey string) (Ticket, error) {
	ticketBytes, err := con.GetStub().GetState(ticketKey)
	if err != nil {
		return Ticket{}, fmt.Errorf("failed to get ticket info from state: %v", err)
	}

	var ticket Ticket
	err = json.Unmarshal(ticketBytes, &ticket)
	if err != nil {
		return Ticket{}, fmt.Errorf("failed to unmarshal ticket from '%s': %v", string(ticketBytes), err)
	}

	return ticket, nil
}
