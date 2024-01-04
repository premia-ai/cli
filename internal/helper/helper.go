package helper

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var PriceDataColumnNames = []string{
	"time",
	"symbol",
	"open",
	"close",
	"high",
	"low",
	"volume",
	"currency",
	"data_provider",
}

type PriceDataRow struct {
	Time         time.Time
	Symbol       string
	Open         string
	Close        string
	High         string
	Low          string
	Volume       string
	Currency     string
	DataProvider string
}

func (p *PriceDataRow) Slice() []any {
	values := make([]any, len(PriceDataColumnNames))
	for idx, columnName := range PriceDataColumnNames {
		switch columnName {
		case "time":
			pgTimestamptz := pgtype.Timestamptz{}
			pgTimestamptz.Time = p.Time
			pgTimestamptz.Valid = true
			values[idx] = pgTimestamptz
		case "open":
			values[idx] = p.Open
		case "close":
			values[idx] = p.Close
		case "high":
			values[idx] = p.High
		case "low":
			values[idx] = p.Low
		case "volume":
			values[idx] = p.Volume
		case "data_provider":
			values[idx] = p.DataProvider
		case "currency":
			values[idx] = p.Currency
		case "symbol":
			values[idx] = p.Symbol
		}
	}

	return values
}

func GetCsvColumn(filePath string, column string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	data, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}

	var columnIdx int
	for i, header := range data[0] {
		if column == header {
			columnIdx = i
			break
		}
	}

	var result []string
	for _, row := range data {
		for i, cell := range row {
			if i == columnIdx {
				result = append(result, cell)
			}
		}
	}

	return result, nil
}

func IsInSlice(slice []string, value string) bool {
	for _, sliceValue := range slice {
		if value == sliceValue {
			return true
		}
	}

	return false
}

func CopyFileToTable(filePath, baseTable string) error {
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

	_, err = conn.Exec(
		context.Background(),
		fmt.Sprintf(
			"COPY %s FROM '%s' DELIMITER ',' CSV HEADER;",
			baseTable,
			filePath,
		),
	)
	if err != nil {
		return err
	}

	return nil
}
