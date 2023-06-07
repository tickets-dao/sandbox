package token

import (
	"fmt"
	"github.com/tickets-dao/foundation/v3/core/types"
	"time"
)

const timeLayout = "2006-01-02 15:04:05"

func (con *Contract) TxReturn(sender *types.Sender, eventID, categoryName, nowString string, row, number int) (TransferEvent, error) {
	lg.Infof("TxReturn start event id: '%s', sender '%s'", eventID, sender.Address())
	now, err := time.Parse(timeLayout, nowString)
	if err != nil {
		return TransferEvent{}, fmt.Errorf("failed to parse now time from '%s': %v", nowString, err)
	}

	issuer, _, err := parseEventID(eventID)
	if err != nil {
		return TransferEvent{}, err
	}

	lg.Infof("issuer '%s', now: '%s'", issuer, now)

	if sender.Equal(issuer) {
		return TransferEvent{}, fmt.Errorf("sender must not be issuer")
	}

	event, err := con.getEventByID(eventID)
	if err != nil {
		err = fmt.Errorf("failed to get event by id: %v", err)
		lg.Error(err)

		return TransferEvent{}, err
	}

	if now.After(event.StartTime.Add(timeThreshold)) {
		//концерт уже прошёл, нельзя погасить
		return TransferEvent{}, fmt.Errorf("now '%s' is after concert start '%s', couldn't burn ticket", now, event.StartTime)
	}

	ticketID := createTicketID(eventID, categoryName, row, number)
	ticket, err := con.getTicket(ticketID)
	if err != nil {
		err = fmt.Errorf("failed to get ticket with id %s: %v", ticketID, err)
		lg.Error(err)

		return TransferEvent{}, err
	}

	lg.Infof("got ticket %#v at '%s'", ticket, ticketID)

	err = con.AllowedBalanceTransfer(rubCurrency, issuer, sender.Address(), ticket.LastBuyPrice, "ticket "+ticketID+" return")
	if err != nil {
		return TransferEvent{}, fmt.Errorf("failed to transfer rubs to issuer: %v", err)
	}

	return con.transferTicket(sender, issuer, ticketID)
}
