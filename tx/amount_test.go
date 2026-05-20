package tx

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type AmountTestSuite struct {
	suite.Suite
}

func TestAmountTestSuite(t *testing.T) {
	suite.Run(t, new(AmountTestSuite))
}

func (s *AmountTestSuite) TestToken() {
	s.Run("constructs from float at given decimals", func() {
		s.Equal(Amount(100000), Token(8, 0.001))
		s.Equal(Amount(150000000), Token(8, 1.5))
		s.Equal(Amount(1500), Token(3, 1.5))
	})

	s.Run("handles zero decimals", func() {
		s.Equal(Amount(42), Token(0, 42))
	})

	s.Run("rounds to nearest to absorb float precision artifacts", func() {
		// 0.1 + 0.2 is 0.30000000000000004 in float64. Naive truncation
		// after *1e8 would give 29999999. math.Round fixes it.
		s.Equal(Amount(30000000), Token(8, 0.1+0.2))
	})
}

func (s *AmountTestSuite) TestDatum() {
	s.Run("constructs from raw base units", func() {
		s.Equal(Amount(123456789), Datum(123456789))
		s.Equal(Amount(0), Datum(0))
	})
}

func (s *AmountTestSuite) TestInt64() {
	s.Equal(int64(150000000), Token(8, 1.5).Int64())
	s.Equal(int64(0), Datum(0).Int64())
}

func (s *AmountTestSuite) TestString() {
	s.Run("returns raw integer with no unit suffix", func() {
		s.Equal("150000000", Token(8, 1.5).String())
		s.Equal("0", Datum(0).String())
		s.Equal("100000", Datum(100000).String())
	})
}

func (s *AmountTestSuite) TestValidate() {
	s.Run("accepts non-negative amounts", func() {
		s.NoError(Amount(0).Validate())
		s.NoError(Amount(100).Validate())
		s.NoError(Datum(1_000_000_000_000).Validate())
	})

	s.Run("rejects negative amounts", func() {
		s.ErrorIs(Amount(-1).Validate(), ErrInvalidAmount)
		s.ErrorIs(Datum(-100).Validate(), ErrInvalidAmount)
	})
}

func (s *AmountTestSuite) TestFormatToken() {
	s.Run("formats fractional amounts with given decimals", func() {
		s.Equal("1.50000000", Token(8, 1.5).FormatToken(8))
		s.Equal("0.00100000", Token(8, 0.001).FormatToken(8))
		s.Equal("1.500", Token(3, 1.5).FormatToken(3))
	})

	s.Run("formats whole numbers with trailing zeros", func() {
		s.Equal("1.00000000", Token(8, 1.0).FormatToken(8))
		s.Equal("100.00000000", Token(8, 100.0).FormatToken(8))
	})

	s.Run("formats sub-unit amounts with leading zeros", func() {
		s.Equal("0.00000001", Datum(1).FormatToken(8))
		s.Equal("0.00000010", Datum(10).FormatToken(8))
	})

	s.Run("returns plain integer when decimals is zero", func() {
		s.Equal("123", Datum(123).FormatToken(0))
		s.Equal("0", Datum(0).FormatToken(0))
	})

	s.Run("handles negative amounts", func() {
		s.Equal("-1.50000000", Token(8, -1.5).FormatToken(8))
		s.Equal("-0.00000001", Datum(-1).FormatToken(8))
	})
}
