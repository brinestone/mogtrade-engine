package contract

import (
	"github.com/brinestone/mogtrade/services/auth"
	"github.com/oklog/ulid/v2"
)

var UlidIdGenerator auth.IdGeneratorFunc = func() string {
	return ulid.Make().String()
}
