package token

import (
	"encoding/json"
	"fmt"
	"github.com/tickets-dao/chaincode/logging"
	"github.com/tickets-dao/foundation/v3/core"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
	"strconv"
	"time"
)

const pricesMapStateSubKey = "prices-map"
const buyBackRateStateSubKey = "buyback-rate"
const issuerBalanceStateSubKey = "balance"
const ticketersStateSubKey = "ticketers"

var lg = logging.NewHTTPLogger("contract")

const issuerAddrString = "5unWkjiVbpAkDDvyS8pxT1hWuwqEFgFShTb8i4WBr2KdDWuuf"

type Contract struct {
	core.BaseContract
	issuer *types.Address
	meta   Metadata
}

// NewContract
func NewContract() *Contract {
	issuer, err := types.AddrFromBase58Check(issuerAddrString)
	if err != nil {
		panic(fmt.Errorf("failed to parse address from '%s': %v", issuerAddrString, err))
	}

	return &Contract{
		BaseContract: core.BaseContract{},
		issuer:       issuer,
		meta: Metadata{
			EventStart:   time.Date(2023, 5, 16, 19, 00, 00, 00, time.Local),
			EventName:    "Лебединое озеро",
			EventAddress: "Театральная площадь, 1",
		},
	}
}

func (con *Contract) GetID() string {
	return "tickets"
}

type PriceCategory struct {
	Name  string
	Seats []Seat
	Price *big.Int
}

type Seat struct {
	Sector int
	Row    int
	Number int
}

func (con *Contract) Issuer() *types.Address {
	return con.issuer
}

func (con *Contract) NBTxPrepare(sender *types.Sender, categoryName string, sector, row, number int, newBurningHash string) error {
	issuerAddress := con.Issuer().String()

	ticketKey := joinStateKey(
		issuerAddress, categoryName, strconv.Itoa(sector), strconv.Itoa(row), strconv.Itoa(number),
	)

	balances, err := con.IndustrialBalanceGet(sender.Address())
	if err != nil {
		return fmt.Errorf("failed to get industrial balances of sender '%s': %v", sender.Address(), err)
	}

	ticketIndustrial, ok := balances[ticketKey]
	if !ok {
		return fmt.Errorf("unathorized for ticket '%s'", ticketKey)
	}

	fmt.Println(ticketIndustrial)

	ticketBytes, err := con.GetStub().GetState(ticketKey)
	if err != nil {
		return fmt.Errorf("failed to get ticket info from state: %v", err)
	}

	if err = con.IndustrialBalanceLock(ticketKey, sender.Address(), new(big.Int).SetInt64(1)); err != nil {
		return fmt.Errorf("failed to lock ticket: %v", err)
	}

	var ticket Ticket
	err = json.Unmarshal(ticketBytes, &ticket)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ticket from '%s': %v", string(ticketBytes), err)
	}

	ticket.BurningHash = newBurningHash

	ticketBytes, _ = json.Marshal(ticket)

	err = con.GetStub().PutState(ticketKey, ticketBytes)
	if err != nil {
		return fmt.Errorf("failed to update ticket: %v", err)
	}

	return nil
}

// QueryMetadata responds with current contract metadata
func (con *Contract) QueryMetadata() (*Metadata, error) {
	return contractMetadata, nil
}

// QueryMetadataTest responds with current contract metadata
func (con *Contract) QueryMetadataTest() (string, error) {
	return `{"hello": "world"}`, nil
}

func (con *Contract) NBTxInitArgsPrint() error {
	argsLen := con.GetInitArgsLen()
	lg.Info("total init args len: ", argsLen)
	for i := 0; i < argsLen; i++ {
		lg.Info(con.GetInitArg(i))
	}

	return nil
}

func (con *Contract) QueryInitArgs(sender *types.Sender) ([]string, error) {
	argsLen := con.GetInitArgsLen()
	lg.SetBackend(logging.NewHTTPBackend())

	initArgs := make([]string, 0, argsLen+2)
	initArgs = append(initArgs, strconv.Itoa(argsLen))
	initArgs = append(initArgs, sender.Address().String())

	lg.Error("total init args len: ", argsLen)
	for i := 0; i < argsLen; i++ {
		lg.Error(con.GetInitArg(i))
		initArgs = append(initArgs, con.GetInitArg(i))
	}

	return initArgs, nil
}

// QueryIndustrialBalanceOf - returns balance of the token for user address
func (con *Contract) QueryIndustrialBalanceOf(address *types.Address) (map[string]string, error) {
	return con.IndustrialBalanceGet(address)
}

// QueryAllowedBalanceOf - returns allowed balance of the token for user address
func (con *Contract) QueryAllowedBalanceOf(address *types.Address, token string) (*big.Int, error) {
	return con.AllowedBalanceGet(token, address)
}

func (con *Contract) createTicketID(categoryName string, sector int, row int, number int) string {
	return fmt.Sprintf("%s::%s::%d::%d::%d",
		con.Issuer().String(), categoryName, sector, row, number,
	)
}
