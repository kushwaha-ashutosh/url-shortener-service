// Package shortener generates short codes for links.
package shortener

import (
	"crypto/rand"
	"math/big"
)

const alphabet = "23456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"

// Generate returns a random base58-ish code of the given length.
// The alphabet excludes visually ambiguous characters (0/O, 1/l/I)
// so codes are safe to read aloud or transcribe.
func Generate(length int) (string, error) {
	code := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for i := range code {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		code[i] = alphabet[n.Int64()]
	}
	return string(code), nil
}
