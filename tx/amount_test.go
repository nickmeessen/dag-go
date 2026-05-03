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

func (s *AmountTestSuite) TestAmount() {
	dagAmount := DAG(1.5)
	s.Equal(int64(150000000), int64(dagAmount))

	datoshiAmount := Datoshi(123456789)
	s.Equal(int64(123456789), int64(datoshiAmount))

	s.Equal(int64(150000000), dagAmount.Int64())

	s.Equal(1.5, dagAmount.DAG())

	s.Contains(dagAmount.String(), "1.50000000 DAG")
	s.Contains(datoshiAmount.String(), "1.23456789 DAG")
}
