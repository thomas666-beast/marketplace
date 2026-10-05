package delivery

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"
)

// crockfordAlphabet is Crockford base32 without I, L, O, U.
// 32 symbols, safe to read over the phone and type on a keypad.
const crockfordAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// cityCodes maps the common names we support to short codes.
// Unknown cities get a deterministic fallback (see CityCode).
var cityCodes = map[string]string{
	"moscow":           "MSK",
	"saint petersburg": "SPB",
	"kazan":            "KZN",
	"novosibirsk":      "NSK",
	"yekaterinburg":    "EKB",
}

// CityCode returns a 3-letter code for a city name.
// Unknown cities get "GEN" (generic) so we never fail on unfamiliar input.
func CityCode(city string) string {
	key := strings.ToLower(strings.TrimSpace(city))
	if code, ok := cityCodes[key]; ok {
		return code
	}
	return "GEN"
}

// GenerateTrackingNumber produces a tracking number in the form:
//
//	CCC-YYYYMMDD-XXXXXX
//
// where CCC is the city code, YYYYMMDD is the UTC date, and XXXXXX
// is six characters of Crockford base32 randomness.
//
// The caller provides `now` so tests are deterministic.
func GenerateTrackingNumber(city string, now time.Time) (string, error) {
	randPart, err := randomString(6)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%s",
		CityCode(city),
		now.UTC().Format("20060102"),
		randPart,
	), nil
}

// GeneratePickupCode produces a 6-character Crockford base32 code.
// This is the code the buyer types at the locker to open the door.
func GeneratePickupCode() (string, error) {
	return randomString(6)
}

// randomString returns n characters from crockfordAlphabet
// using crypto/rand for unpredictability.
func randomString(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("randomString: n must be positive")
	}
	out := make([]byte, n)
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("randomString: %w", err)
	}
	for i, b := range buf {
		out[i] = crockfordAlphabet[int(b)%len(crockfordAlphabet)]
	}
	return string(out), nil
}
