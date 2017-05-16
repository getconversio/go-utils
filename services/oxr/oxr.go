package oxr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/getconversio/go-utils/util"
)

const (
	endpoint  = "https://openexchangerates.org/api"
	userAgent = "goutils/0.1.0"
)

var (
	cache Cache
	token string
)

type Cache interface {
	Add(string, Rates) error
	Get(string) (*Rates, error)
}

type Rates struct {
	ID        string             `json:"id" bson:"_id"`
	License   string             `json:"license" bson:"license"`
	Timestamp int                `json:"timestamp" bson:"timestamp"`
	Base      string             `json:"base" bson:"base"`
	Rates     map[string]float64 `json:"rates" bson:"rates"`
}

type NotFoundError struct{}

func (e NotFoundError) Error() string {
	return "not found"
}

// Returns historical rates for the given date string, formatted as "yyyy-mm-dd"
func Historical(date string) (*Rates, error) {
	u := fmt.Sprintf("%s/historical/%s.json?app_id=%s", endpoint, date, token)
	resp, err := http.Get(u)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, errors.New(fmt.Sprintf("%s: %s", http.StatusText(resp.StatusCode), body))
	}
	rates := Rates{}
	err = json.NewDecoder(resp.Body).Decode(&rates)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Unable to parse JSON: %s", err))
	}
	return &rates, nil
}

func cachedHistoricalRates(date time.Time) (*Rates, error) {
	dateStr := date.Format("2006-01-02")
	rates, err := cache.Get(dateStr)
	if err == nil {
		return rates, nil
	} else if _, ok := err.(NotFoundError); !ok {
		return nil, err
	}

	rates, err = Historical(dateStr)
	if err != nil {
		return nil, err
	}

	rates.ID = dateStr
	cache.Add(rates.ID, *rates)
	return rates, nil
}

// Convert an amount between two currencies. Uses historical exchange rates for the given timestamp.
func ConvertCurrency(from, to string, amount float64, date time.Time) (float64, error) {
	rates, err := cachedHistoricalRates(date)
	if err != nil || rates == nil {
		return 0, err
	}

	fromRate, okFrom := rates.Rates[from]
	toRate, okTo := rates.Rates[to]
	if okFrom && okTo {
		return amount / fromRate * toRate, nil
	}

	return 0, errors.New(fmt.Sprintf("Cannot convert from %v to %v", from, to))
}

// Setup options for
type Options struct {
	Token string
	Cache Cache
}

// Prepares for Open Exchange Rates API calls. Includes setting up the exchange
// rate cache. Expects environment variables such as
func Setup(options *Options) {
	if options == nil {
		options = &Options{}
	}

	if options.Token == "" {
		// Try an environment variable for OXR if the given options token is empty.
		options.Token = os.Getenv("OPEN_EXCHANGE_RATES")
	}

	if options.Cache == nil {
		var err error
		options.Cache, err = NewLRUCache(100)
		util.PanicOnError("Could not initialize LRU Cache", err)
	}

	token = options.Token
	cache = options.Cache
}
