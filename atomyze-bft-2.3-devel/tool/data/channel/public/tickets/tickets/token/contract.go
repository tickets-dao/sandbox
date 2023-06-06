package token

import (
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
const eventsInfoStateKey = "events"

const timeThreshold = 3 * time.Hour

var lg = logging.NewHTTPLogger("contract")

const issuerAddrString = "5unWkjiVbpAkDDvyS8pxT1hWuwqEFgFShTb8i4WBr2KdDWuuf"

type Contract struct {
	core.BaseContract
	meta Metadata
}

// NewContract
func NewContract() *Contract {
	return &Contract{
		BaseContract: core.BaseContract{},
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
	Name  string   `json:"name"`
	Rows  int      `json:"rows"`
	Seats int      `json:"seats"`
	Price *big.Int `json:"price"`
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

func createTicketID(eventID, categoryName string, row, number int) string {
	return fmt.Sprintf("%s::%s::%d::%d", eventID, categoryName, row, number)
}
