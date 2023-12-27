package twelvedata

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/premia-ai/cli/internal/dataprovider"
	"github.com/premia-ai/cli/internal/helper"
)

type Timespan string

const (
	Minute Timespan = "min"
	Hour   Timespan = "h"
	Day    Timespan = "day"
	Week   Timespan = "week"
	Month  Timespan = "month"
)

type Instrument struct {
	MetaData   MetaData          `json:"meta"`
	TimeSeries []TimeSeriesValue `json:"values"`
}

type MetaData struct {
	Symbol           string
	Interval         string
	Currency         string
	ExchangeTimezone string `json:"exchange_timezone"`
	Exchange         string
	Type             string
}

type TimeSeriesValue struct {
	DateTime string
	Open     string
	Close    string
	High     string
	Low      string
	Volume   string
}

func DownloadCandles(apiParams *dataprovider.ApiParams, filePath string) error {
	var seedFile *os.File
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		seedFile, err = os.Create(filePath)
		if err != nil {
			return err
		}
	} else {
		seedFile, err = os.OpenFile(filePath, os.O_APPEND, 0600)
		if err != nil {
			return err
		}
	}
	defer seedFile.Close()

	aggregates, err := getAggregates(apiParams)
	if err != nil {
		return err
	}

	writer := csv.NewWriter(seedFile)
	defer writer.Flush()

	err = writer.Write(helper.StocksCsvColumns)
	if err != nil {
		return err
	}

	for _, instrument := range aggregates {
		for _, timeSeriesValue := range instrument.TimeSeries {
			row := []string{
				timeSeriesValue.DateTime,
				instrument.MetaData.Symbol,
				timeSeriesValue.Open,
				timeSeriesValue.Close,
				timeSeriesValue.High,
				timeSeriesValue.Low,
				timeSeriesValue.Volume,
				instrument.MetaData.Currency,
				string(dataprovider.TwelveData),
			}
			err = writer.Write(row)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getAggregates(apiParams *dataprovider.ApiParams) (map[string]Instrument, error) {
	apiKey := os.Getenv("TWELVEDATA_API_KEY")
	if apiKey == "" {
		// TODO: Set up an alternative to enter the API key in the CLI via
		// a password input
		log.Fatal("Please set TWELVEDATA_API_KEY environment variable")
	}

	// Format for interval needs to be: "1min", "1h", "1day", "1week", "1month"
	timespan, err := mapTimespan(apiParams.Timespan)
	if err != nil {
		return nil, err
	}
	interval := fmt.Sprintf("%d%s", apiParams.Quantity, timespan)

	query := url.Values{}
	query.Set("apikey", apiKey)
	query.Set("interval", interval)
	query.Set("symbol", strings.Join(apiParams.Tickers, ","))
	// Format: 2023-12-01 00:00:00
	query.Set("start_date", apiParams.From.Format(time.DateTime))
	// Format: 2023-12-01 00:00:00
	query.Set("end_date", apiParams.To.Format(time.DateTime))
	query.Set("timezone", "UTC")
	query.Set("format", "JSON")

	url := url.URL{
		Scheme:   "https",
		Host:     "api.twelvedata.com",
		Path:     "time_series",
		RawQuery: query.Encode(),
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var responseBody map[string]Instrument
	err = json.Unmarshal(body, &responseBody)
	if err != nil {
		fmt.Fprint(os.Stderr, "body: ", string(body), "\n")
		return nil, err
	}

	return responseBody, nil
}

func mapTimespan(timespan dataprovider.Timespan) (Timespan, error) {
	switch timespan {
	case dataprovider.Minute:
		return Minute, nil
	case dataprovider.Hour:
		return Hour, nil
	case dataprovider.Day:
		return Day, nil
	case dataprovider.Week:
		return Week, nil
	case dataprovider.Month:
		return Month, nil
	default:
		return "", errors.New(
			fmt.Sprintf(
				"Timespan '%d' is not supported by twelvedata",
				timespan,
			),
		)
	}
}
