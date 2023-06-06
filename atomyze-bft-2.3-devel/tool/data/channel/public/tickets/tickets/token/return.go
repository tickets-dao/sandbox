package token

import (
	"fmt"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
	"time"
)

func (con *Contract) NBTxReturn(sender *types.Sender, eventID, categoryName, nowString string, row, number int) (TransferEvent, error) {
	lg.Infof("TxBuy start event id: '%s'", eventID)
	now, err := time.Parse(time.RFC3339, nowString)
	if err != nil {
		return TransferEvent{}, fmt.Errorf("failed to parse now time from '%s': %v", nowString, err)
	}

	issuer, _, err := parseEventID(eventID)
	if err != nil {
		return TransferEvent{}, err
	}

	if sender.Equal(issuer) {
		return TransferEvent{}, fmt.Errorf("sender must not be issuer")
	}

	event, err := con.getEventByID(eventID)
	if err != nil {
		return TransferEvent{}, fmt.Errorf("failed to get event by id: %v", err)
	}

	if now.After(event.StartTime.Add(timeThreshold)) {
		//концерт уже прошёл, нельзя погасить
		return TransferEvent{}, fmt.Errorf("now '%s' is after concert start '%s', couldn't burn ticket", now, event.StartTime)
	}

	ticketID := createTicketID(eventID, categoryName, row, number)
	err = con.AllowedBalanceTransfer(rubCurrency, issuer, sender.Address(), big.NewInt(1), "ticket "+ticketID+" return")
	if err != nil {
		return TransferEvent{}, fmt.Errorf("failed to transfer rubs to issuer: %v", err)
	}

	return con.transferTicket(sender, issuer, eventID, categoryName, row, number)
}
