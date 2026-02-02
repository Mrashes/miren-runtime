package joincode

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

const alphanumChars = "abcdefghjkmnpqrstuvwxyz23456789"

var codePattern = regexp.MustCompile(`^[a-z]+-[a-z]+-[a-z0-9]{4}$`)

func Generate() (string, error) {
	adjIdx, err := randomIndex(len(adjectives))
	if err != nil {
		return "", fmt.Errorf("selecting adjective: %w", err)
	}

	nounIdx, err := randomIndex(len(nouns))
	if err != nil {
		return "", fmt.Errorf("selecting noun: %w", err)
	}

	suffix, err := randomAlphanumeric(4)
	if err != nil {
		return "", fmt.Errorf("generating suffix: %w", err)
	}

	return fmt.Sprintf("%s-%s-%s", adjectives[adjIdx], nouns[nounIdx], suffix), nil
}

func Hash(code string) string {
	normalized := strings.ToLower(strings.TrimSpace(code))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func Validate(code string) bool {
	return codePattern.MatchString(strings.ToLower(strings.TrimSpace(code)))
}

func randomIndex(max int) (int, error) {
	buf := make([]byte, 1)
	_, err := rand.Read(buf)
	if err != nil {
		return 0, err
	}
	return int(buf[0]) % max, nil
}

func randomAlphanumeric(length int) (string, error) {
	result := make([]byte, length)
	for i := range result {
		buf := make([]byte, 1)
		_, err := rand.Read(buf)
		if err != nil {
			return "", err
		}
		result[i] = alphanumChars[int(buf[0])%len(alphanumChars)]
	}
	return string(result), nil
}
