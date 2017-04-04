package geoip

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

var endpoint = os.Getenv("GEOIP_URL")

type GeoipResponse struct {
	IP          string  `json:"ip"`
	CountryCode string  `json:"country_code"`
	CountryName string  `json:"country_name"`
	RegionCode  string  `json:"region_code"`
	RegionName  string  `json:"region_name"`
	City        string  `json:"city"`
	Zip         string  `json:"zip_code"`
	MetroCode   uint    `json:"metro_code"`
	TimeZone    string  `json:"time_zone"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

func Lookup(ip string) (*GeoipResponse, error) {
	u := fmt.Sprintf("%s/json/%s", endpoint, ip)
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, errors.New(fmt.Sprintf("%s: %s", http.StatusText(resp.StatusCode), body))
	}
	gip := GeoipResponse{}
	err = json.NewDecoder(resp.Body).Decode(&gip)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Unable to parse JSON: %s", err))
	}
	return &gip, nil
}
