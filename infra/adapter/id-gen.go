package adapter

import (
	"github.com/brinestone/mogtrade/core/contract"
	"github.com/oklog/ulid/v2"
)

var UlidIdGenerator contract.IdFactory = func() string {
	return ulid.Make().String()
}
