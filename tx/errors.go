package tx

import "errors"

// ErrInvalidAmount is returned by Amount.Validate and wallet.Sign when an
// Amount fails its non-negativity invariant.
var ErrInvalidAmount = errors.New("amount must be non-negative")
