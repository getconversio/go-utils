package oxr

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/jarcoal/httpmock.v1"
)

const oxrRates string = `{
    "disclaimer": "The disclaimer",
    "license": "The license",
    "timestamp": 1464771613,
    "base": "USD",
    "rates": {
        "DKK": 7,
        "SEK": 8,
        "USD": 1
    }
}`

func setupOXR() {
	Setup()
	httpmock.Activate()
}

func teardownOXR() {
	httpmock.DeactivateAndReset()
}

func TestConvertCurrency(t *testing.T) {
	setupOXR()
	defer teardownOXR()

	httpCallCount := 0
	httpmock.RegisterResponder("GET", "https://openexchangerates.org/api/historical/2016-01-01.json",
		func(req *http.Request) (*http.Response, error) {
			httpCallCount++
			return httpmock.NewStringResponse(200, oxrRates), nil
		})

	jan1 := time.Date(2016, time.January, 1, 0, 0, 0, 0, time.UTC)
	amount, err := ConvertCurrency("SEK", "DKK", 56.0, jan1)
	assert.NoError(t, err)
	assert.Equal(t, 49.0, amount)
	assert.Equal(t, 1, httpCallCount)

	amount, err = ConvertCurrency("SEK", "USD", 56.0, jan1)
	assert.NoError(t, err)
	assert.Equal(t, 7.0, amount)

	// The currencies should be cached on the second conversion.
	assert.Equal(t, 1, httpCallCount)

	// Error on unknown currencies.
	_, err = ConvertCurrency("USD", "NOK", 1.0, jan1)
	assert.EqualError(t, err, "Cannot convert from USD to NOK")
}
