package polygon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	polygon "github.com/polygon-io/client-go/rest"
	"github.com/polygon-io/client-go/rest/iter"
	"github.com/polygon-io/client-go/rest/models"
	"github.com/premia-ai/cli/internal/dataprovider"
)

const currency = "USD"

// TODO: Extend polygon to allow for multiple tickers
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

	timespan, err := mapTimespan(apiParams.Timespan)
	if err != nil {
		return err
	}

	aggregates := getAggregates(&models.ListAggsParams{
		Ticker:     apiParams.Tickers[0],
		From:       models.Millis(apiParams.From),
		To:         models.Millis(apiParams.To),
		Timespan:   timespan,
		Multiplier: apiParams.Quantity,
	})

	header := fmt.Sprintln(
		"time,symbol,open,close,high,low,volumee,data_provider",
	)

	_, err = seedFile.WriteString(header)
	if err != nil {
		return err
	}

	for aggregates.Next() {
		item := aggregates.Item()
		data := fmt.Sprintf(
			"%s,%s,%f,%f,%f,%f,%d,%s,%s\n",
			time.Time(item.Timestamp).Format(time.RFC3339),
			apiParams.Tickers[0],
			item.Open,
			item.Close,
			item.High,
			item.Low,
			int(item.Volume),
			currency,
			dataprovider.Polygon,
		)

		_, err := seedFile.WriteString(data)

		if err != nil {
			return err
		}
	}
	if aggregates.Err() != nil {
		return aggregates.Err()
	}

	return nil
}

func getAggregates(apiParams *models.ListAggsParams) *iter.Iter[models.Agg] {
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
