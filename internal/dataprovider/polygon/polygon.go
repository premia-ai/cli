package polygon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	polygon "github.com/polygon-io/client-go/rest"
	"github.com/polygon-io/client-go/rest/iter"
	"github.com/polygon-io/client-go/rest/models"
	"github.com/premia-ai/cli/internal/dataprovider"
	"github.com/premia-ai/cli/internal/helper"
)

type RowSrcMeta struct {
	iter   *iter.Iter[models.Agg]
	ticker string
}

type RowSrc struct {
	meta   RowSrcMeta
	values []any
	err    error
}

func (r *RowSrc) Next() bool {
	next := r.meta.iter.Next()
	if next == false {
		return false
	}

	if r.meta.iter.Err() != nil {
		r.err = r.meta.iter.Err()
		return false
	}

	item := r.meta.iter.Item()
	row := helper.PriceDataRow{
		Time:         time.Time(item.Timestamp),
		Open:         strconv.FormatFloat(item.Open, 'f', -1, 64),
		Close:        strconv.FormatFloat(item.Close, 'f', -1, 64),
		High:         strconv.FormatFloat(item.High, 'f', -1, 64),
		Low:          strconv.FormatFloat(item.Low, 'f', -1, 64),
		Volume:       strconv.FormatInt(int64(item.Volume), 10),
		Currency:     currency,
		DataProvider: string(dataprovider.Polygon),
		Symbol:       r.meta.ticker,
	}

	r.values = row.Slice()
	return true
}

func (r *RowSrc) Values() ([]any, error) {
	return r.values, r.err
}

func (r *RowSrc) Err() error {
	return r.err
}

func NewRowSrc(ticker string, value *iter.Iter[models.Agg]) *RowSrc {
	return &RowSrc{
		meta: RowSrcMeta{
			iter:   value,
			ticker: ticker,
		},
	}
}

const currency = "USD"

// TODO: Extend polygon to allow for multiple tickers
func ImportStocks(apiParams *dataprovider.ApiParams) error {
	timespan, err := mapTimespan(apiParams.Timespan)
	if err != nil {
		return err
	}

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

	candles := getStockCandles(&models.ListAggsParams{
		Ticker:     apiParams.Tickers[0],
		From:       models.Millis(apiParams.From),
		To:         models.Millis(apiParams.To),
		Timespan:   timespan,
		Multiplier: apiParams.Quantity,
	})

	_, err = conn.CopyFrom(
		context.Background(),
		pgx.Identifier{apiParams.Table},
		helper.PriceDataColumnNames,
		NewRowSrc(apiParams.Tickers[0], candles),
	)
	if err != nil {
		return err
	}

	return nil
}

func getStockCandles(apiParams *models.ListAggsParams) *iter.Iter[models.Agg] {
	polygon_api_key := os.Getenv("POLYGON_API_KEY")
	if polygon_api_key == "" {
		// TODO: Set up an alternative to enter the API key in the CLI via
		// a password input
		log.Fatal("Please set POLYGON_API_KEY environment variable")
	}

	return polygon.
		New(polygon_api_key).
		ListAggs(context.Background(), apiParams)
}

func mapTimespan(timespan dataprovider.Timespan) (models.Timespan, error) {
	switch timespan {
	case dataprovider.Second:
		return models.Second, nil
	case dataprovider.Minute:
		return models.Minute, nil
	case dataprovider.Hour:
		return models.Hour, nil
	case dataprovider.Day:
		return models.Day, nil
	case dataprovider.Week:
		return models.Week, nil
	case dataprovider.Month:
		return models.Month, nil
	case dataprovider.Quarter:
		return models.Quarter, nil
	case dataprovider.Year:
		return models.Year, nil
	default:
		return "", errors.New(
			fmt.Sprintf("Timespan '%d' is not supported by polygon", timespan),
		)
	}
}
