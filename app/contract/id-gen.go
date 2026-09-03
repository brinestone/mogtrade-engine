package contract

import (
	"github.com/brinestone/mogtrade/core/encoding"
	"github.com/oklog/ulid/v2"
)

var UlidIdGenerator encoding.IdGeneratorFunc = func() string {
	return ulid.Make().String()
}
