package delivery_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/thomas666-beast/marketplace/internal/delivery"
)

func TestCityCode_Known(t *testing.T) {
	cases := map[string]string{
		"Moscow":           "MSK",
		"moscow":           "MSK",
		"  MOSCOW  ":       "MSK",
		"Saint Petersburg": "SPB",
		"Kazan":            "KZN",
	}
	for input, want := range cases {
		t.Run(input, func(t *testing.T) {
			require.Equal(t, want, delivery.CityCode(input))
		})
	}
}

func TestCityCode_Unknown(t *testing.T) {
	require.Equal(t, "GEN", delivery.CityCode("Atlantis"))
	require.Equal(t, "GEN", delivery.CityCode(""))
}

func TestGenerateTrackingNumber_Format(t *testing.T) {
	now := time.Date(2026, 10, 4, 15, 30, 0, 0, time.UTC)
	tn, err := delivery.GenerateTrackingNumber("Moscow", now)
	require.NoError(t, err)

	// Expected: MSK-20261004-XXXXXX
	pattern := regexp.MustCompile(`^MSK-20261004-[0-9A-HJKMNP-TV-Z]{6}$`)
	require.Regexp(t, pattern, tn, "got: %s", tn)
}

func TestGenerateTrackingNumber_UnknownCity(t *testing.T) {
	now := time.Date(2026, 10, 4, 15, 30, 0, 0, time.UTC)
	tn, err := delivery.GenerateTrackingNumber("Atlantis", now)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(tn, "GEN-20261004-"),
		"unknown city should use GEN prefix, got: %s", tn)
}

func TestGenerateTrackingNumber_NoAmbiguousChars(t *testing.T) {
	now := time.Now()
	// Generate many, ensure no I, L, O, U appear in the random part.
	for i := 0; i < 100; i++ {
		tn, err := delivery.GenerateTrackingNumber("Moscow", now)
		require.NoError(t, err)

		parts := strings.Split(tn, "-")
		require.Len(t, parts, 3)
		randomPart := parts[2]

		for _, c := range randomPart {
			require.NotContains(t, "ILOU", string(c),
				"Crockford alphabet must exclude I, L, O, U; found %c in %s", c, tn)
		}
	}
}

func TestGenerateTrackingNumber_Uniqueness(t *testing.T) {
	now := time.Now()
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		tn, err := delivery.GenerateTrackingNumber("Moscow", now)
		require.NoError(t, err)
		require.False(t, seen[tn], "collision detected: %s", tn)
		seen[tn] = true
	}
}

func TestGeneratePickupCode_Format(t *testing.T) {
	code, err := delivery.GeneratePickupCode()
	require.NoError(t, err)
	require.Len(t, code, 6)

	pattern := regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{6}$`)
	require.Regexp(t, pattern, code, "got: %s", code)
}

func TestGeneratePickupCode_Uniqueness(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		code, err := delivery.GeneratePickupCode()
		require.NoError(t, err)
		require.False(t, seen[code], "collision detected: %s", code)
		seen[code] = true
	}
}

func TestStatus_IsTerminal(t *testing.T) {
	terminal := []delivery.Status{
		delivery.StatusPickedUp,
		delivery.StatusReturned,
		delivery.StatusCancelled,
	}
	for _, s := range terminal {
		require.True(t, s.IsTerminal(), "%s should be terminal", s)
	}

	nonTerminal := []delivery.Status{
		delivery.StatusCreated,
		delivery.StatusAwaitingDispatch,
		delivery.StatusInTransit,
		delivery.StatusArrivedAtHub,
		delivery.StatusOutForDelivery,
		delivery.StatusReadyForPickup,
	}
	for _, s := range nonTerminal {
		require.False(t, s.IsTerminal(), "%s should not be terminal", s)
	}
}
