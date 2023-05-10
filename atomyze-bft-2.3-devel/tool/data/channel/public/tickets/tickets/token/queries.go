package token

import (
	"fmt"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
	"sort"
	"strconv"
	"strings"
	"time"
)

const eventID = "42"

type Event struct {
	StartTime time.Time `json:"start_time"`
	Address   string    `json:"address"`
	Name      string    `json:"name"`
	ID        string    `json:"id"`
}

// QueryIndustrialBalanceOf - returns balance of the token for user address
func (con *Contract) QueryIndustrialBalanceOf(address *types.Address) (map[string]string, error) {
	return con.IndustrialBalanceGet(address)
}

// QueryAllowedBalanceOf - returns allowed balance of the token for user address
func (con *Contract) QueryAllowedBalanceOf(address *types.Address, token string) (*big.Int, error) {
	return con.AllowedBalanceGet(token, address)
}

// QueryIndustrialBalanceOf - returns balance of the token for user address
func (con *Contract) QueryEvents() ([]Event, error) {
	return []Event{
		{
			StartTime: time.Date(2023, 5, 16, 19, 00, 00, 00, time.Local),
			Address:   "Театральная площадь, 1",
			Name:      "Лебединое озеро",
			ID:        eventID,
		},
	}, nil
}

// QueryIndustrialBalanceOf - returns balance of the token for user address
func (con *Contract) QueryEventsByIDs(eventIDs string) ([]Event, error) {

	return []Event{
		{
			StartTime: time.Date(2023, 5, 16, 19, 00, 00, 00, time.Local),
			Address:   "Театральная площадь, 1",
			Name:      "Лебединое озеро",
			ID:        eventID,
		},
	}, nil
}

// QueryEventCategories - returns all categories for event
func (con *Contract) QueryEventCategories(eventID string) ([]string, error) {
	pricesMap, err := con.getPricesMap()
	if err != nil {
		return nil, err
	}

	var categories = make([]string, 0, len(pricesMap))
	for category := range pricesMap {
		categories = append(categories, category)
	}

	sort.Slice(categories, func(i, j int) bool {
		return categories[i] <= categories[j]
	})

	return categories, err
}

// QueryTicketsByCategory - returns all categories for event
func (con *Contract) QueryTicketsByCategory(eventID, category string) ([]Ticket, error) {
	pricesMap, err := con.getPricesMap()
	if err != nil {
		return nil, err
	}

	price, ok := pricesMap[category]
	if !ok {
		return nil, fmt.Errorf("category '%s' is not present in event categories: %v", category, pricesMap)
	}

	availableTickets, err := con.IndustrialBalanceGet(con.Issuer())
	if err != nil {
		return nil, err
	}

	tickets := make([]Ticket, 0, len(availableTickets))
	for ticket := range availableTickets {
		ticketParts := strings.Split(ticket, "::")
		if ticketParts[1] != category {
			continue
		}

		ticketFromKey, err := ticketFromKeyParts(ticketParts)
		if err != nil {
			lg.Errorf("failed to parse ticket from key '%s': %v", ticket, err)
			continue
		}

		ticketFromKey.Price = int32(price.Int64())

		tickets = append(tickets, ticketFromKey)
	}

	sort.Slice(tickets, func(i, j int) bool {
		return tickets[i].String() <= tickets[j].String()
	})

	return tickets, err
}

func (con *Contract) QueryMyTickets(sender *types.Sender) ([]Ticket, error) {
	senderAddress := sender.Address()
	lg.Infof("query my tickets for sender '%s'", senderAddress)
	ticketsStringsMap, err := con.IndustrialBalanceGet(senderAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get industrial balance: %v", err)
	}

	myTickets := make([]Ticket, 0, len(ticketsStringsMap))

	for ticketString := range ticketsStringsMap {
		ticket, err := ticketFromKeyParts(strings.Split(ticketString, "::"))
		if err != nil {
			lg.Errorf("failed to parse ticket fom string: '%s': %v", err)
		}

		ticket.Owner = senderAddress.String()

		myTickets = append(myTickets, ticket)
	}

	sort.Slice(myTickets, func(i, j int) bool {
		return myTickets[i].String() <= myTickets[j].String()
	})

	lg.Infof("query myTickets for sender '%s' done, got %d tickets", senderAddress, len(myTickets))

	return myTickets, nil
}

func ticketFromKeyParts(keyParts []string) (Ticket, error) {
	if len(keyParts) != 5 {
		return Ticket{}, fmt.Errorf("expected %d parts in key, got %d", 5, len(keyParts))
	}

	sector, err := strconv.ParseInt(keyParts[2], 10, 32)
	if err != nil {
		return Ticket{}, fmt.Errorf("failed to parse sector: %v", err)
	}

	row, err := strconv.ParseInt(keyParts[3], 10, 32)
	if err != nil {
		return Ticket{}, fmt.Errorf("failed to parse row: %v", err)
	}

	number, err := strconv.ParseInt(keyParts[4], 10, 32)
	if err != nil {
		return Ticket{}, fmt.Errorf("failed to parse number: %v", err)
	}

	return Ticket{
		Category: keyParts[1],
		Sector:   int(sector),
		Row:      int(row),
		Number:   int(number),
		EventID:  eventID,
	}, nil
}
