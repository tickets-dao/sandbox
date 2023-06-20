package token

import (
	"github.com/tickets-dao/foundation/v3/core/types/big"
	"strconv"
)

// TransferEvent ...
type TransferEvent struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Price  int64  `json:"price"`     // Цена, которую To заплатил за трансфер, если возврат, то цена будет отрицательной
	Ticker string `json:"ticket_id"` // Уникальное название тикера билета
}

// PrepareEvent событие, которое создается при подготовке билета для прохода на мероприятие
type PrepareEvent struct {
	Owner       string `json:"from"`
	Ticker      string `json:"ticket_id"` // Уникальное название тикера билета
	BurningHash string `json:"burning_hash"`
}

// BurnEvent событие, которое создается при сжигании билета
type BurnEvent struct {
	Owner       string `json:"from"`
	Ticketer    string `json:"ticketer"`
	Ticker      string `json:"ticket_id"` // Уникальное название тикера билета
	BurningHash string `json:"burning_hash"`
}

type ConfirmEvent struct {
	Verifier          string `json:"verifier"`           // Verifier хранит адрас верифаера, подтвердившего мероприятие
	ConsensusGathered bool   `json:"consensus_gathered"` // ConsensusGathered показывает собран ли консенсус после этого голоса
}

type Ticket struct {
	BurningHash string `json:"burning_hash"`
	// стоимость последней покупки этого билета, используется для расчета стоимости возврата билета
	LastBuyPrice *big.Int `json:"last_buy_price,omitempty"`
	// текущий владелец билета
	Owner string `json:"owner"`

	Category string `json:"category"`
	Row      int    `json:"row"`
	Number   int    `json:"number"`
	EventID  string `json:"event_id"`
}

func (t Ticket) String() string {
	return joinStateKey(t.EventID, t.Category, strconv.Itoa(t.Row), strconv.Itoa(t.Number))
}
