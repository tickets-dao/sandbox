package token

import (
	"fmt"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
	"strconv"
	"strings"
	"time"
)

const showID = "42"

type Show struct {
	startTime time.Time
	address   string
	name      string
	id        string
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
func (con *Contract) QueryShows() ([]Show, error) {
	return []Show{
		{
			startTime: time.Date(2023, 5, 16, 19, 00, 00, 00, time.Local),
			address:   "Театральная площадь, 1",
			name:      "Лебединое озеро",
			id:        showID,
		},
	}, nil
}

// QueryCategories - returns all categories for show
func (con *Contract) QueryCategories(showID string) ([]string, error) {
	pricesMap, err := con.getPricesMap()
	if err != nil {
		return nil, err
	}

	var categories = make([]string, 0, len(pricesMap))
	for category := range pricesMap {
		categories = append(categories, category)
	}

	return categories, err
}

// QueryTicketsByCategory - returns all categories for show
func (con *Contract) QueryTicketsByCategory(category string) ([]Ticket, error) {
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

		tickets = append(tickets, ticketFromKey)
	}

	return tickets, err
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

	number, err := strconv.ParseInt(keyParts[3], 10, 32)
	if err != nil {
		return Ticket{}, fmt.Errorf("failed to parse number: %v", err)
	}

	return Ticket{
		Category: keyParts[1],
		Sector:   int(sector),
		Row:      int(row),
		Number:   int(number),
		ShowID:   showID,
	}, nil

}
