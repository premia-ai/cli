package dataprovider

import "time"

type Provider string

const (
	Polygon    Provider = "polygon.io"
	TwelveData Provider = "twelvedata.com"
	Csv        Provider = "csv"
)

type Timespan int

const (
	Second Timespan = iota
	Minute
	Hour
	Day
	Week
	Month
	Quarter
	Year
)

type TimespanInfo struct {
	OneLetterCode string
	Value         Timespan
	Unit          string
	BiggerUnits   []string
}

var Timespans = []TimespanInfo{
	{
		OneLetterCode: "s",
		Value:         Second,
		Unit:          "second",
		BiggerUnits:   []string{"minute", "hour", "day", "week"},
	},
	{
		OneLetterCode: "m",
		Value:         Minute,
		Unit:          "minute",
		BiggerUnits:   []string{"hour", "day", "week"},
	},
	{
		OneLetterCode: "h",
		Value:         Hour,
		Unit:          "hour",
		BiggerUnits:   []string{"day", "week"},
	},
	{
		OneLetterCode: "d",
		Value:         Day,
		Unit:          "day",
		BiggerUnits:   []string{"week"},
	},
	{
		OneLetterCode: "w",
		Value:         Week,
		Unit:          "week",
		BiggerUnits:   []string{},
	},
}

type ApiParams struct {
	Tickers  []string
	Timespan Timespan
	Quantity int
	From     time.Time
	To       time.Time
	Table    string
}
