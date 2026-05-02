package tx

import "fmt"

// Amount represents a quantity of DAG denominated in datoshi (1 DAG = 10^8
// datoshi). It is the canonical on-wire unit — all balances, transfers, and
// fees in dag-go are expressed as Amount. The underlying int64 comfortably
// holds every valid on-chain value (max DAG supply ~5×10^17 datoshi vs int64
// max ~9.2×10^18).
type Amount int64

// DAG constructs an Amount from a whole-DAG value. The fractional part is
// truncated at datoshi precision (10^-8 DAG); float rounding at this scale
// is acceptable for user-facing input. Callers needing exact values should
// use Datoshi.
func DAG(dag float64) Amount {
	return Amount(dag * 1e8)
}

// Datoshi constructs an Amount directly from raw datoshi. Use when precision
// matters — for example, forwarding a balance straight into a transfer.
func Datoshi(n int64) Amount {
	return Amount(n)
}

// Int64 returns the Amount as raw datoshi.
func (a Amount) Int64() int64 {
	return int64(a)
}

// DAG returns the Amount as a whole-DAG floating-point value. Suitable for
// display; not suitable for arithmetic.
func (a Amount) DAG() float64 {
	return float64(a) / 1e8
}

// String formats the Amount as "N.NNNNNNNN DAG" with full datoshi precision.
func (a Amount) String() string {
	return fmt.Sprintf("%.8f DAG", a.DAG())
}
