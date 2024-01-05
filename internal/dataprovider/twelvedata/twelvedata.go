package twelvedata

import (
	"context"
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

	"github.com/jackc/pgx/v5"
	"github.com/premia-ai/cli/internal/dataprovider"
	"github.com/premia-ai/cli/internal/helper"
)

const apiTimestamp = "2006-01-02 15:04:05"

type Timespan string

const (
	Minute Timespan = "min"
	Hour   Timespan = "h"
	Day    Timespan = "day"
	Week   Timespan = "week"
	Month  Timespan = "month"
)

type ApiResponse struct {
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

type RowSrc struct {
	idx    int
	meta   []helper.MarketDataRow
	values []any
	err    error
}

func (r *RowSrc) Next() bool {
	if r.idx > len(r.meta)-1 {
		return false
	}

	item := r.meta[r.idx]
	r.idx += 1

	r.values = item.Slice()
	return true
}

func (r *RowSrc) Values() ([]any, error) {
	return r.values, r.err
}

func (r *RowSrc) Err() error {
	return r.err
}

func NewRowSrc(value []helper.MarketDataRow) *RowSrc {
	return &RowSrc{
		meta: value,
	}
}

func ImportMarketData(apiParams *dataprovider.ApiParams) error {
	postgresUrl := os.Getenv("POSTGRES_URL")
	if postgresUrl == "" {
		return errors.New("Please set POSTGRES_URL environment variable")
	}

	conn, err := pgx.Connect(context.Background(), postgresUrl)
	if err != nil {
		return errors.New(fmt.Sprintf(
			"Unable to connect to database: %v\n", err,
		))
	}
	defer conn.Close(context.Background())

	candles, err := getAggregates(apiParams)
	if err != nil {
		return err
	}

	_, err = conn.CopyFrom(
		context.Background(),
		pgx.Identifier{apiParams.Table},
		helper.MarketDataColumnNames,
		NewRowSrc(candles),
	)
	if err != nil {
		return err
	}

	return nil
}

func getAggregates(apiParams *dataprovider.ApiParams) ([]helper.MarketDataRow, error) {
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

	var responseBody map[string]ApiResponse
	err = json.Unmarshal(body, &responseBody)
	if err != nil {
		fmt.Fprint(os.Stderr, "body: ", string(body), "\n")
		return nil, err
	}

	var values []helper.MarketDataRow
	for _, instrument := range responseBody {
		for _, timeSeriesValue := range instrument.TimeSeries {
			t, err := time.Parse(apiTimestamp, timeSeriesValue.DateTime)
			if err != nil {
				return nil, err
			}

			values = append(values, helper.MarketDataRow{
				Time:         t,
				Symbol:       instrument.MetaData.Symbol,
				Open:         timeSeriesValue.Open,
				Close:        timeSeriesValue.Close,
				High:         timeSeriesValue.High,
				Low:          timeSeriesValue.Low,
				Volume:       timeSeriesValue.Volume,
				Currency:     instrument.MetaData.Currency,
				DataProvider: string(dataprovider.TwelveData),
			})
		}
	}

	return values, nil
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
