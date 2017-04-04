package geoip

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/jarcoal/httpmock.v1"
)

func setup() {
	httpmock.Activate()
}

func teardown() {
	httpmock.DeactivateAndReset()
}

func TestLookup(t *testing.T) {
	setup()
	defer teardown()

	httpmock.RegisterResponder("GET", "https://example.com/json/8.8.8.8",
		httpmock.NewStringResponder(200, `{"ip":"8.8.8.8","country_code":"US","country_name":"United States",
			"region_code":"CA","region_name":"California","city":"Mountain View","zip_code":"94035",
			"time_zone":"America/Los_Angeles","latitude":37.386,"longitude":-122.0838,"metro_code":807}`))

	httpmock.RegisterResponder("GET", "https://example.com/json/1.1.1.1",
		httpmock.NewStringResponder(200, "{"))

	httpmock.RegisterResponder("GET", "https://example.com/json/2.2.2.2",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, ""), errors.New("Something else went wrong")
		})

	httpmock.RegisterResponder("GET", "https://example.com/json/3.3.3.3",
		httpmock.NewStringResponder(503, "I don't know"))

	geoIp, err := Lookup("8.8.8.8")
	assert.NoError(t, err)
	assert.Equal(t, &GeoipResponse{
		"8.8.8.8",
		"US",
		"United States",
		"CA",
		"California",
		"Mountain View",
		"94035",
		807,
		"America/Los_Angeles",
		37.386,
		-122.0838,
	}, geoIp)

	geoIp, err = Lookup("1.1.1.1")
	assert.Nil(t, geoIp)
	assert.EqualError(t, err, "Unable to parse JSON: unexpected EOF")

	geoIp, err = Lookup("2.2.2.2")
	assert.Nil(t, geoIp)
	assert.EqualError(t, err, "Get https://example.com/json/2.2.2.2: Something else went wrong")

	geoIp, err = Lookup("3.3.3.3")
	assert.Nil(t, geoIp)
	assert.EqualError(t, err, "Service Unavailable: I don't know")
}
