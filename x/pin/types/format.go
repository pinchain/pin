package types

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

const (
	// PINAlphabet is the set of characters a PIN may contain: A-Z and 2-9 with the
	// visually ambiguous characters 0, 1, I, O and L removed.
	PINAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

	// PINGroupLen is the length of the first group of a canonical PIN.
	PINGroupLen = 3
	// PINSuffixLen is the length of the second group of a canonical PIN.
	PINSuffixLen = 4

	// PINSeparator separates the two PIN groups.
	PINSeparator = "-"

	// PINLength is the length of a canonical PIN, including the separator.
	PINLength = PINGroupLen + len(PINSeparator) + PINSuffixLen
)

// pinRegexp matches a canonical PIN: XXX-XXXX over the PIN alphabet.
var pinRegexp = regexp.MustCompile(
	fmt.Sprintf(`^[%s]{%d}%s[%s]{%d}$`, PINAlphabet, PINGroupLen, PINSeparator, PINAlphabet, PINSuffixLen),
)

// NormalizePIN converts user input into the canonical PIN representation:
// surrounding whitespace is dropped and letters are uppercased. No other
// rewriting happens, so structurally invalid input stays invalid.
func NormalizePIN(pin string) string {
	return strings.ToUpper(strings.TrimSpace(pin))
}

// ValidatePIN reports whether pin is a canonical PIN.
func ValidatePIN(pin string) error {
	if pin == "" {
		return ErrInvalidPIN.Wrap("pin is empty")
	}
	if !pinRegexp.MatchString(pin) {
		return ErrInvalidPIN.Wrapf(
			"pin %q is not in canonical %s form over alphabet %s",
			pin, strings.Repeat("X", PINGroupLen)+PINSeparator+strings.Repeat("X", PINSuffixLen), PINAlphabet,
		)
	}
	return nil
}

// NormalizeAndValidatePIN normalizes user input and validates the result.
func NormalizeAndValidatePIN(pin string) (string, error) {
	normalized := NormalizePIN(pin)
	if err := ValidatePIN(normalized); err != nil {
		return "", err
	}
	return normalized, nil
}

// GeneratePIN returns a new random canonical PIN. Randomness comes from
// crypto/rand only: PINs are never derived from timestamps, counters, block
// height or any other predictable value.
func GeneratePIN() (string, error) {
	var sb strings.Builder
	sb.Grow(PINLength)
	limit := big.NewInt(int64(len(PINAlphabet)))
	for i := 0; i < PINGroupLen+PINSuffixLen; i++ {
		if i == PINGroupLen {
			sb.WriteString(PINSeparator)
		}
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", fmt.Errorf("failed to read secure randomness: %w", err)
		}
		sb.WriteByte(PINAlphabet[n.Int64()])
	}
	pin := sb.String()
	if err := ValidatePIN(pin); err != nil {
		// Unreachable unless the alphabet and the regexp disagree.
		return "", err
	}
	return pin, nil
}
