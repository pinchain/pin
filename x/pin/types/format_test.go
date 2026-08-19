package types_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pinchain/pinchain/x/pin/types"
)

func TestNormalizePIN(t *testing.T) {
	require.Equal(t, "A7K-92XM", types.NormalizePIN("a7k-92xm"))
	require.Equal(t, "A7K-92XM", types.NormalizePIN("  A7k-92Xm \n"))
	require.Equal(t, "B4M-89QZ", types.NormalizePIN("b4m-89qz"))
}

func TestValidatePIN(t *testing.T) {
	valid := []string{"A7K-92XM", "B4M-89QZ", "ZZZ-9999", "222-2222"}
	for _, pin := range valid {
		require.NoError(t, types.ValidatePIN(pin), pin)
	}

	invalid := map[string]string{
		"empty":                             "",
		"lowercase":                         "a7k-92xm",
		"missing separator":                 "A7K92XM",
		"wrong group length":                "A7-92XM",
		"too long":                          "A7K-92XMM",
		"excluded I":                        "A7I-92XM",
		"excluded O":                        "A7O-92XM",
		"excluded L":                        "A7L-92XM",
		"excluded 0":                        "A70-92XM",
		"excluded 1":                        "A71-92XM",
		"brief example B4M-81QZ contains 1": "B4M-81QZ",
		"symbols":                           "A7K-92X!",
		"whitespace inside":                 "A7K -92XM",
		"unicode":                           "A7K-92XÅ",
	}
	for name, pin := range invalid {
		require.ErrorIs(t, types.ValidatePIN(pin), types.ErrInvalidPIN, name)
	}
}

func TestNormalizeAndValidatePIN(t *testing.T) {
	pin, err := types.NormalizeAndValidatePIN(" a7k-92xm ")
	require.NoError(t, err)
	require.Equal(t, "A7K-92XM", pin)

	_, err = types.NormalizeAndValidatePIN("a7i-92xm")
	require.ErrorIs(t, err, types.ErrInvalidPIN)
}

func TestGeneratePIN(t *testing.T) {
	seen := make(map[string]struct{}, 512)
	for i := 0; i < 512; i++ {
		pin, err := types.GeneratePIN()
		require.NoError(t, err)
		require.NoError(t, types.ValidatePIN(pin))
		require.Equal(t, types.PINLength, len(pin))
		require.Equal(t, pin, types.NormalizePIN(pin))
		seen[pin] = struct{}{}
	}
	// The keyspace is 31^7, so 512 draws must not collide in practice; a
	// counter- or timestamp-derived generator would fail this.
	require.Len(t, seen, 512)

	// No excluded characters may ever appear.
	for pin := range seen {
		for _, ch := range strings.Replace(pin, types.PINSeparator, "", 1) {
			require.Contains(t, types.PINAlphabet, string(ch))
		}
	}
}

func TestPINAlphabetExcludesAmbiguousCharacters(t *testing.T) {
	for _, excluded := range []string{"0", "1", "I", "O", "L"} {
		require.NotContains(t, types.PINAlphabet, excluded)
	}
}
