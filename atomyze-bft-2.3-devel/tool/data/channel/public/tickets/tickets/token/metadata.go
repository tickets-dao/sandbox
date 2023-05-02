package token

import (
	"time"

	"github.com/tickets-dao/foundation/v3/core/types"
)

const minVerifiersVotesCount = 2

var contractMetadata *Metadata

// Metadata информация о концерте
type Metadata struct {
	EventStart   time.Time
	EventName    string
	EventAddress string
	Issuer       *types.Address
	Verifiers    []*types.Address
}
