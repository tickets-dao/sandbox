package token

import (
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
	"sort"
	"strconv"
	"strings"
	"time"
)

const defaultEventID = "42"

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

// QueryEvents - returns list of all events
func (con *Contract) QueryEvents() ([]Event, error) {
	iterator, err := con.GetStub().GetStateByPartialCompositeKey(eventsInfoStateKey, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to get iterator: %v", err)
	}

	defer func(iterator shim.StateQueryIteratorInterface) {
		err2 := iterator.Close()
		if err2 != nil {
			lg.Errorf("got error %v on closing iterator", err2)
		}
	}(iterator)

	var eventsInfo []Event
	var totalCount int

	for iterator.HasNext() {
		totalCount++
		info, err := iterator.Next()
		if err != nil {
			lg.Errorf("failed to get next event from iterator: %v", err)
			continue
		}

		var issuerInfo Event
		err = json.Unmarshal(info.Value, &issuerInfo)
		if err != nil {
			lg.Errorf("failed to unmarshal event from '%s': %v", string(info.Value), err)
			continue
		}

		eventsInfo = append(eventsInfo, issuerInfo)
	}

	sort.Slice(eventsInfo, func(i, j int) bool {
		return eventsInfo[i].Name <= eventsInfo[j].Name
	})

	lg.Infof("got %d events info from %d total count", len(eventsInfo), totalCount)

	return eventsInfo, nil
}

// QueryEventsByIDs - returns list of events
func (con *Contract) QueryEventsByIDs(eventIDsString string) ([]Event, error) {
	var eventIDs []string
	err := json.Unmarshal([]byte(eventIDsString), &eventIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal event ids from '%s': %v", eventIDsString, err)
	}

	lg.Infof("get events with ids '%v'", eventIDs)

	events := make([]Event, 0, len(eventIDs))
	for _, eventID := range eventIDs {
		event, err := con.getEventByID(eventID)
		if err != nil {
			lg.Errorf("failed to get event with id '%s': %v", eventID, err)
			continue
		}

		events = append(events, event)
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].ID <= events[j].ID
	})

	return events, nil
}

// QueryEventsByIssuer - returns list of events per issuer
func (con *Contract) QueryEventsByIssuer(address *types.Address) ([]Event, error) {
	lg.Infof("starting query events by issuer with address: '%s'", address)

	iterator, err := con.GetStub().GetStateByPartialCompositeKey(eventsInfoStateKey, []string{address.String()})
	if err != nil {
		return nil, fmt.Errorf("failed to get iterator: %v", err)
	}

	defer func(iterator shim.StateQueryIteratorInterface) {
		err2 := iterator.Close()
		if err2 != nil {
			lg.Errorf("got error %v on closing iterator", err2)
		}
	}(iterator)

	var events []Event
	var totalCount int

	for iterator.HasNext() {
		totalCount++
		info, err := iterator.Next()
		if err != nil {
			lg.Errorf("failed to get next info from iterator: %v", err)
			continue
		}

		var event Event
		err = json.Unmarshal(info.Value, &event)
		if err != nil {
			lg.Errorf("failed to unmarshal event from '%s': %v", string(info.Value), err)
			continue
		}

		events = append(events, event)
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].ID <= events[j].ID
	})

	lg.Infof("got %d events from %d total count", len(events), totalCount)

	return events, nil
}

// QueryEventCategories - returns all categories for event
func (con *Contract) QueryEventCategories(eventID string) ([]PriceCategory, error) {
	pricesMap, err := con.getPricesMap(eventID)
	if err != nil {
		return nil, err
	}

	var categories = make([]PriceCategory, 0, len(pricesMap))
	for category := range pricesMap {
		categories = append(categories, PriceCategory{Name: category, Price: pricesMap[category]})
	}

	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Name <= categories[j].Name
	})

	return categories, err
}

// QueryTicketsByCategory - returns all categories for event
func (con *Contract) QueryTicketsByCategory(eventID, category string) ([]Ticket, error) {
	issuer, _, err := parseEventID(eventID)
	if err != nil {
		return nil, err
	}

	pricesMap, err := con.getPricesMap(eventID)
	if err != nil {
		return nil, err
	}

	price, ok := pricesMap[category]
	if !ok {
		return nil, fmt.Errorf("category '%s' is not present in event categories: %v", category, pricesMap)
	}

	availableTickets, err := con.IndustrialBalanceGet(issuer)
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
		Row:      int(row),
		Number:   int(number),
		EventID:  joinStateKey(keyParts[0], keyParts[1]),
	}, nil
}

func (con Contract) getEventByID(eventID string) (Event, error) {
	lg.Infof("get event with id '%s'", eventID)
	address, eventNum, err := parseEventID(eventID)
	if err != nil {
		return Event{}, err
	}

	eventInfoKey, err := con.GetStub().CreateCompositeKey(eventsInfoStateKey, []string{address.String(), strconv.Itoa(eventNum)})
	if err != nil {
		return Event{}, fmt.Errorf("failed to create composite key for saving event's info: %v", err)
	}

	eventBytes, err := con.GetStub().GetState(eventInfoKey)
	if err != nil {
		return Event{}, fmt.Errorf("failed to get state at '%s': %v", eventInfoKey, err)
	}

	var event Event
	if err = json.Unmarshal(eventBytes, &event); err != nil {
		return Event{}, fmt.Errorf("failed to unmarshal event from '%s': %v", string(eventBytes), err)
	}

	return event, nil
}
