package tx

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/suite"
)

type EncodingTestSuite struct {
	suite.Suite
}

func TestEncodingTestSuite(t *testing.T) {
	suite.Run(t, new(EncodingTestSuite))
}

// referenceTransfer is the same fixture used in wallet_test.go TestSign,
// where the entire signing chain (Encode → Kryo → SHA → ECDSA) is
// cross-validated against dag4.js v2.8.1. The intermediate outputs locked
// in below are therefore byte-for-byte protocol-correct; these tests guard
// against accidental edits to the encoding primitives.
var referenceTransfer = Transfer{
	Source:      "DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy",
	Destination: "DAG1ATvdAxGz4DNzPdrk6p8QD1CdSXNLvYypaaVQ",
	Amount:      Datoshi(100000),
	Fee:         Datoshi(10000),
	Parent: Ref{
		Hash:    "0000000000000000000000000000000000000000000000000000000000000000",
		Ordinal: 0,
	},
	Salt: 8725724278030335,
}

func (s *EncodingTestSuite) TestEncode() {
	s.Run("matches reference vector", func() {
		want := "240DAG3jifKUZPc213rRLSfZVSLZfPRfX7fwTGh8tsy40DAG1ATvdAxGz4DNzPdrk6p8QD1CdSXNLvYypaaVQ5186a064000000000000000000000000000000000000000000000000000000000000000010510000141effffffffffff"
		s.Equal(want, referenceTransfer.Encode())
	})
}

func (s *EncodingTestSuite) TestKryoSerialize() {
	s.Run("wraps reference encode output with prefix and length", func() {
		want, err := hex.DecodeString("03f602323430444147336a69664b555a506332313372524c53665a56534c5a665052665837667754476838747379343044414731415476644178477a34444e7a5064726b367038514431436453584e4c7659797061615651353138366130363430303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030313035313030303031343165666666666666666666666666")
		s.Require().NoError(err)
		s.Equal(want, KryoSerialize(referenceTransfer.Encode()))
	})

	s.Run("empty input still emits prefix and length-of-1", func() {
		// utf8Length(1) → [0x81], so output is [0x03, 0x81]
		s.Equal([]byte{0x03, 0x81}, KryoSerialize(""))
	})
}

func (s *EncodingTestSuite) TestUtf8Length() {
	cases := []struct {
		name  string
		input int
		want  []byte
	}{
		{"branch 1 lower (0)", 0, []byte{0x80}},
		{"branch 1 upper (63)", 63, []byte{0xbf}},
		{"branch 2 lower (64)", 64, []byte{0xc0, 0x01}},
		{"branch 2 upper (8191)", 8191, []byte{0xff, 0x7f}},
		{"branch 3 lower (8192)", 8192, []byte{0xc0, 0x80, 0x01}},
		{"branch 3 upper (1048575)", 1048575, []byte{0xff, 0xff, 0x7f}},
		{"branch 4 lower (1048576)", 1048576, []byte{0xc0, 0x80, 0x80, 0x01}},
		{"branch 4 upper (134217727)", 134217727, []byte{0xff, 0xff, 0xff, 0x7f}},
		{"branch 5 lower (134217728)", 134217728, []byte{0xc0, 0x80, 0x80, 0x80, 0x01}},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.Equal(tc.want, utf8Length(tc.input))
		})
	}
}

func (s *EncodingTestSuite) TestGenerateSalt() {
	s.Run("output always within [minSalt, minSalt+2^48)", func() {
		const upper = minSalt + (1 << 48)
		for i := 0; i < 1000; i++ {
			n, err := GenerateSalt()
			s.Require().NoError(err)
			s.GreaterOrEqual(n, minSalt)
			s.Less(n, upper)
		}
	})

	s.Run("1000 calls yield 1000 distinct values (entropy sanity)", func() {
		seen := make(map[uint64]struct{}, 1000)
		for i := 0; i < 1000; i++ {
			n, err := GenerateSalt()
			s.Require().NoError(err)
			seen[n] = struct{}{}
		}
		s.Len(seen, 1000, "expected no collisions across 1000 fresh salts")
	})
}
