package oxr

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/jarcoal/httpmock.v1"
	"gopkg.in/mgo.v2"
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

func setup() {
	Setup(nil)
	httpmock.Activate()
}

func teardown() {
	httpmock.DeactivateAndReset()
}

func TestBasicSetup(t *testing.T) {
	Setup(nil)

	// It should fetch the api key from the environment by default
	assert.Equal(t, "apikey", token)

	// It should use LRU cache by default
	assert.IsType(t, &LRUCache{}, cache)
}

func testConvertCurrency(t *testing.T) {
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

func TestConvertCurrency(t *testing.T) {
	setup()
	defer teardown()
	testConvertCurrency(t)
}

func TestConvertCurrencyWithMongo(t *testing.T) {
	setup()
	defer teardown()

	session, err := mgo.Dial(os.Getenv("MONGODB_URL"))
	require.NoError(t, err)
	session.DB("").C("oxr").DropCollection()
	session.DB("").C("blah").DropCollection()

	// Test default collection
	Setup(&Options{
		Cache: NewMongoCache(session, "", ""),
	})
	testConvertCurrency(t)

	// Test specific collection
	Setup(&Options{
		Cache: NewMongoCache(session, "", "blah"),
	})
	testConvertCurrency(t)

	// Just to make sure we actually used MongoDB :-)
	cnt, err := session.DB("").C("oxr").Find(nil).Count()
	require.NoError(t, err)
	assert.Equal(t, 1, cnt)
	cnt, err = session.DB("").C("blah").Find(nil).Count()
	require.NoError(t, err)
	assert.Equal(t, 1, cnt)
}
