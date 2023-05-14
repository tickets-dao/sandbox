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

// NBTxEmission - create tickets emission
func (con *Contract) NBTxEmission(sender *types.Sender, priceCategoriesString, name, address, eventTimeString string) error {
	lg.Infof("this is new emission for sender %s\n", sender.Address().String())

	issuerInfo, err := con.getIssuerInfo(sender.Address())
	if err != nil {
		return fmt.Errorf("failed to get issuerInfo")
	}

	eventID := createEventID(sender.Address(), issuerInfo.NextEventID)
	issuerInfo.NextEventID++

	if err = con.saveIssuerInfo(sender.Address(), issuerInfo); err != nil {
		return fmt.Errorf("failed to save issuer info: %v", err)
	}

	if err = con.saveEventInfo(eventID, name, address, eventTimeString); err != nil {
		return err
	}

	lg.Infof("price categories: %s\n", priceCategoriesString)

	var priceCategories []PriceCategory
	if err = json.Unmarshal([]byte(priceCategoriesString), &priceCategories); err != nil {
		return fmt.Errorf("failed to unmarshal price categories: %v", err)
	}

	if err = con.saveCategoriesMap(eventID, priceCategories); err != nil {
		return err
	}

	stub := con.GetStub()

	if err = stub.PutState(joinStateKey(eventID, ticketersStateSubKey), []byte("[]")); err != nil {
		return fmt.Errorf("failed to save ticketers addreses: %v", err)
	}

	if err = stub.PutState(joinStateKey(eventID, issuerBalanceStateSubKey), []byte("0")); err != nil {
		return fmt.Errorf("failed to save ticketers addreses: %v", err)
	}

	if err = stub.PutState(joinStateKey(eventID, buyBackRateStateSubKey), []byte("1")); err != nil {
		return fmt.Errorf("failed to save ticketers addreses: %v", err)
	}

	if err = con.emitTickets(sender.Address(), eventID, priceCategories); err != nil {
		return err
	}

	lg.Infof("all done, returning nil error\n")

	return nil
}

func (con *Contract) saveCategoriesMap(eventID string, categories []PriceCategory) error {
	categoriesMap := make(map[string]*big.Int, len(categories))
	for _, category := range categories {
		if _, ok := categoriesMap[category.Name]; ok {
			return fmt.Errorf("category '%s' is used more than once", category.Name)
		}

		categoriesMap[category.Name] = category.Price
	}

	categoriesMapBytes, err := json.Marshal(categoriesMap)
	if err != nil {
		return fmt.Errorf("failed to marshal categories map: %v", err)
	}

	lg.Infof("categories map: '%s'\n", string(categoriesMapBytes))

	stub := con.GetStub()
	if err = stub.PutState(joinStateKey(eventID, pricesMapStateSubKey), categoriesMapBytes); err != nil {
		return fmt.Errorf("failed to save categories map state: %v", err)
	}

	return nil
}

func (con *Contract) getPricesMap(eventID string) (map[string]*big.Int, error) {
	pricesMapBytes, err := con.GetStub().GetState(joinStateKey(eventID, pricesMapStateSubKey))
	if err != nil {
		return nil, fmt.Errorf("failed to get prices map: %v", err)
	}

	var pricesMap map[string]*big.Int

	if err = json.Unmarshal(pricesMapBytes, &pricesMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal prices map: %v", err)
	}

	return pricesMap, nil
}

func (con *Contract) emitTickets(issuer *types.Address, eventID string, categories []PriceCategory) error {

	// билеты, которые нужно выпустить, соответствует ключу в стейте
	ticketKeys := make([]string, 0, 10*len(categories))
	var err error

	for _, category := range categories {
		if category.Rows > 100 {
			return fmt.Errorf("category %s has more than 100 rows (%d)", category.Name, category.Rows)
		}
		if category.Seats > 100 {
			return fmt.Errorf("category %s has more than 100 seats in a row (%d)", category.Name, category.Seats)
		}

		for row := 1; row <= category.Rows; row++ {
			for seat := 1; seat <= category.Seats; seat++ {
				ticketID := con.createTicketID(eventID, category.Name, row, seat)
				ticketKeys = append(ticketKeys, ticketID)
			}
		}
	}

	lg.Infof("got %d tickets", len(ticketKeys))
	for _, ticketID := range ticketKeys {
		err = con.IndustrialBalanceAdd(ticketID, issuer, new(big.Int).SetInt64(1), "initial emission")
		if err != nil {
			return fmt.Errorf("failed to emit ticket %s: %v", ticketID, err)
		}
	}

	return nil
}

func createEventID(address *types.Address, eventNum int) string {
	return joinStateKey(address.String(), strconv.Itoa(eventNum))
}

func mustParseEventID(eventID string) (string, int) {
	parts := strings.Split(eventID, "::")
	if len(parts) != 2 {
		panic(fmt.Sprintf("expected 2 parts in event id, got %d: '%s'", len(parts), eventID))
	}

	eventNum, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil {
		panic(fmt.Sprintf("failed to parse event num from '%s': %v", parts[1], err))
	}

	return parts[0], int(eventNum)
}

func (con *Contract) saveEventInfo(eventID string, eventName string, address string, eventTimeString string) error {
	eventTime, err := time.Parse("2006-01-02 15:04:05", eventTimeString)
	if err != nil {
		return fmt.Errorf("failed to parse event time from '%s': %v", eventTimeString, err)
	}

	eventBytes, err := json.Marshal(Event{
		StartTime: eventTime,
		Address:   address,
		Name:      eventName,
		ID:        eventID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal event bytes: %v", err)
	}

	issuer, eventNum, err := parseEventID(eventID)
	if err != nil {
		return fmt.Errorf("failed to parse event id '%s': %v", eventID, err)
	}

	eventInfoKey, err := con.GetStub().CreateCompositeKey(eventsInfoStateKey, []string{issuer.String(), strconv.Itoa(eventNum)})
	if err != nil {
		return fmt.Errorf("failed to create composite key for saving event's info: %v", err)
	}

	if err = con.GetStub().PutState(eventInfoKey, eventBytes); err != nil {
		return fmt.Errorf("failed to save event's info: %v", err)
	}

	return nil
}
