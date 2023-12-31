package migrations

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/premia-ai/cli/internal/config"
	"github.com/premia-ai/cli/internal/dataprovider"
	"github.com/premia-ai/cli/internal/dataprovider/polygon"
	"github.com/premia-ai/cli/internal/dataprovider/twelvedata"
	"github.com/premia-ai/cli/internal/helper"
	"github.com/premia-ai/cli/resource"
)

type SqlTemplateData struct {
	InstrumentType config.InstrumentType
	Quantity       int
	TimeUnit       string
	ReferenceTable string
}

var migrationVersion = 1

func getFeatureNames() ([]string, error) {
	templateExtension := ".up.template.sql"
	entries, err := resource.Fs.ReadDir(resource.TemplateFeaturesPath)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		// Only check one suffix to avoid duplicates
		if !strings.HasSuffix(entry.Name(), templateExtension) {
			continue
		}

		names = append(
			names, strings.Replace(entry.Name(), templateExtension, "", 1),
		)
	}

	return names, nil
}

// TODO: Implement dry-run
// TODO: Implement verbose
func Initialize() error {
	postgresUrl := os.Getenv("POSTGRES_URL")
	if postgresUrl == "" {
		return errors.New("Please set POSTGRES_URL environment variable")
	}

	migrationsDir, err := config.MigrationsDir(true)

	err = CreateMigration(
		"add_timescale",
		SqlTemplateData{},
	)
	if err != nil {
		return err
	}

	add_stocks, err := askBoolQuestion("Do you want to store stock price data?")

	if err != nil {
		return err
	}

	if add_stocks == false {
		return nil
	}

	err = addInstrumentMigrations(config.Stocks)
	if err != nil {
		return err
	}

	add_options, err := askBoolQuestion(
		"Do you want to store option price data?",
	)

	if err != nil {
		return err
	}

	if add_options == false {
		return nil
	}

	err = addInstrumentMigrations(config.Options)
	if err != nil {
		return err
	}

	err = applyMigrations(migrationsDir, postgresUrl)
	if err != nil {
		return err
	}

	return nil
}

func addInstrumentMigrations(instrumentType config.InstrumentType) error {
	var timespanUnits []string
	for _, timespan := range dataprovider.Timespans {
		timespanUnits = append(timespanUnits, timespan.Unit)
	}

	// Create raw data table migration
	timespanUnit, err := askSelectQuestion(
		"What is the timespan of your data points?",
		timespanUnits,
	)
	if err != nil {
		return err
	}

	var timespan dataprovider.TimespanInfo
	for _, f := range dataprovider.Timespans {
		if f.Unit == timespanUnit {
			timespan = f
		}
	}

	err = CreateMigration(
		"add_candles",
		SqlTemplateData{
			InstrumentType: instrumentType,
			Quantity:       1,
			TimeUnit:       timespan.Unit,
		},
	)
	if err != nil {
		return err
	}

	// Create aggregate table
	// TODO: The user should be able to create multiple aggregate tables
	baseTable := fmt.Sprintf(
		"%s_1_%s_candles",
		instrumentType,
		timespan.Unit,
	)

	err = config.UpdateConfig(
		instrumentType,
		&config.InstrumentConfig{
			BaseTable:    baseTable,
			TimespanUnit: timespan.Unit,
		},
	)
	if err != nil {
		return err
	}

	switch instrumentType {
	case config.Stocks:
		err = CreateMigration(
			"add_companies",
			SqlTemplateData{},
		)
	case config.Options:
		err = CreateMigration(
			"add_contracts",
			SqlTemplateData{},
		)
	}
	if err != nil {
		return err
	}

	addAggregate, err := askBoolQuestion(
		"Do you want to create an aggregate based on your raw data?",
	)

	if addAggregate {
		aggregateTimespanUnit, err := askSelectQuestion(
			"Which duration should the table have?",
			timespan.BiggerUnits,
		)
		if err != nil {
			return err
		}

		var aggregateTimespanInfo dataprovider.TimespanInfo
		for _, f := range dataprovider.Timespans {
			if f.Unit == aggregateTimespanUnit {
				aggregateTimespanInfo = f
			}
		}

		err = CreateMigration(
			"add_aggregate_candles",
			SqlTemplateData{
				InstrumentType: instrumentType,
				Quantity:       1,
				TimeUnit:       aggregateTimespanInfo.Unit,
				ReferenceTable: baseTable,
			},
		)
		if err != nil {
			return err
		}
	}

	// Create feature table
	// TODO: The user should be able to create multiple feature tables
	addFeature, err := askBoolQuestion(
		"Do you want to create a feature table based on your raw data?",
	)
	if err != nil {
		return err
	}

	if addFeature {
		featureNames, err := getFeatureNames()
		if err != nil {
			return err
		}

		featureName, err := askSelectQuestion(
			"Which feature would you like to add?",
			featureNames,
		)
		if err != nil {
			return err
		}

		err = CreateMigration(
			featureName,
			SqlTemplateData{
				InstrumentType: instrumentType,
				Quantity:       1,
				TimeUnit:       timespan.Unit,
				ReferenceTable: baseTable,
			},
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// TODO: Make Seed work with both stocks and options, right now it works just
// with stocks
// TODO: Check which tables have been initialized instead of clumsily failing
func Seed() error {
	configData, err := config.Config()
	if err != nil {
		return err
	}
	// TODO: Move this check out of the function
	shouldSeedDb, err := askBoolQuestion("Would you like to seed the database?")
	if err != nil {
		return err
	}

	var timespan dataprovider.TimespanInfo
	for _, f := range dataprovider.Timespans {
		// TODO: There needs to be a fix to avoid failure when stocks tables
		// are not set up
		if f.Unit == configData.Instruments[config.Stocks].TimespanUnit {
			timespan = f
		}
	}

	// TODO: Move this to a Seed function and call it from the cmd directly
	if shouldSeedDb {
		dataProviders := []string{
			string(dataprovider.Polygon),
			string(dataprovider.TwelveData),
			string(dataprovider.Csv),
		}

		provider, err := askSelectQuestion(
			"Which method would you like to use to seed the database?",
			dataProviders,
		)
		if err != nil {
			return err
		}

		if provider == dataProviders[0] {
			ticker, err := askInputQuestion(
				"What is the ticker of the equity you would like to download?",
			)
			if err != nil {
				return err
			}

			from, err := askInputQuestion(
				"What should the start date of the entries be?",
			)
			if err != nil {
				return err
			}
			fromTime, err := time.Parse(time.RFC3339, from)
			if err != nil {
				return err
			}

			to, err := askInputQuestion(
				"What should the end date of the entries be?",
			)
			if err != nil {
				return err
			}
			toTime, err := time.Parse(time.RFC3339, to)
			if err != nil {
				return err
			}

			err = polygon.ImportMarketData(&dataprovider.ApiParams{
				Tickers:  []string{ticker},
				From:     fromTime,
				To:       toTime,
				Timespan: timespan.Value,
				Quantity: 1,
				Table:    configData.Instruments[config.Stocks].BaseTable,
			})
			if err != nil {
				return err
			}

			// TODO: don't allow twelvedata when the timespan is in seconds
		} else if provider == dataProviders[1] {
			shouldUseCsv, err := askBoolQuestion("Do you want to use a CSV file to select tickers for seeding?")
			if err != nil {
				return err
			}

			var tickers []string
			if shouldUseCsv {
				tickersFilePath, err := askInputQuestion("What is the path to your csv file?")
				if err != nil {
					return err
				}

				column, err := askInputQuestion("What is the column name for the tickers?")

				tickers, err = helper.GetCsvColumn(tickersFilePath, column)
				if err != nil {
					return err
				}
			} else {
				result, err := askInputQuestion(
					"Which tickers would you like to download? (separate values by ,)",
				)
				if err != nil {
					return err
				}
				tickers = strings.Split(result, ",")
			}

			from, err := askInputQuestion(
				"What should the start date of the entries be?",
			)
			if err != nil {
				return err
			}
			fromTime, err := time.Parse(time.RFC3339, from)
			if err != nil {
				return err
			}

			to, err := askInputQuestion(
				"What should the end date of the entries be?",
			)
			if err != nil {
				return err
			}
			toTime, err := time.Parse(time.RFC3339, to)
			if err != nil {
				return err
			}

			err = twelvedata.ImportMarketData(&dataprovider.ApiParams{
				Tickers:  tickers,
				Timespan: timespan.Value,
				Quantity: 1,
				From:     fromTime,
				To:       toTime,
				Table:    configData.Instruments[config.Stocks].BaseTable,
			})
			if err != nil {
				return err
			}
		} else if provider == dataProviders[2] {
			seedFilePath, err := askInputQuestion(
				"What is the path to your CSV file?",
			)
			if err != nil {
				return err
			}

			// TODO: There needs to be a fix to avoid failure when stocks tables
			// are not set up
			err = helper.CopyFileToTable(seedFilePath, configData.Instruments[config.Stocks].BaseTable)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func applyMigrations(migrationsPath string, databaseUrl string) error {
	// file:// needs to be added otherwise the New method is throwing an error
	m, err := migrate.New("file://"+migrationsPath, databaseUrl)
	if err != nil {
		return err
	}

	err = m.Up()
	if err == migrate.ErrNoChange {
		// TODO: This should only be displayed in verbose mode
		fmt.Fprintln(os.Stderr, "No migration was applied.")
	} else if err != nil {
		return err
	} else {
		fmt.Println("Successfully applied migrations.")
	}

	return nil
}

func askBoolQuestion(question string) (bool, error) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(question + "\n 1 yes\n 2 no\n\n>> ")

		scanner.Scan()
		response := scanner.Text()

		if response == "y" || response == "1" || response == "yes" {
			return true, nil
		}
		if response == "n" || response == "2" || response == "no" {
			return false, nil
		}
		if scanner.Err() != nil {
			return false, scanner.Err()
		}
	}
}

func askSelectQuestion(question string, options []string) (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		var optionString string
		for i, option := range options {
			optionString += fmt.Sprintf(" %d %s\n", i+1, option)
		}
		fmt.Printf(question + "\n" + optionString + "\n>> ")

		scanner.Scan()
		response := scanner.Text()
		if helper.IsInSlice(options, response) {
			return response, nil
		}

		// User used the number to identify element
		responseInt, err := strconv.Atoi(response)
		if err != nil {
			continue
		}
		if responseInt <= len(options) {
			return options[responseInt-1], nil
		}

		if scanner.Err() != nil {
			return "", scanner.Err()
		}
	}
}

func askInputQuestion(question string) (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf(question + "\n\n>> ")

		scanner.Scan()
		response := scanner.Text()
		if response != "" {
			return response, nil
		}
		if scanner.Err() != nil {
			return "", scanner.Err()
		}
	}
}

func isInSlice(slice []string, value string) bool {
	for _, sliceValue := range slice {
		if value == sliceValue {
			return true
		}
	}

	return false
}

func getMigrationName(templateName string, version int) string {
	return fmt.Sprintf(
		"%d_%s",
		version,
		strings.Replace(templateName, ".template", "", 1),
	)
}

func CreateMigration(templateName string, data SqlTemplateData) error {
	err := createPartialMigration(
		templateName+".up.template.sql",
		migrationVersion,
		data,
	)
	if err != nil {
		return err
	}

	err = createPartialMigration(
		templateName+".down.template.sql",
		migrationVersion,
		data,
	)

	migrationVersion += 1
	return err
}

func createPartialMigration(
	templateName string,
	version int,
	data SqlTemplateData,
) error {
	funcMap := template.FuncMap{
		"sub": func(a, b int) int { return a - b },
	}
	migration, err := template.New(templateName).Funcs(funcMap).ParseFS(
		resource.Fs, path.Join("*/*", templateName),
	)
	if err != nil {
		return err
	}

	migrationName := getMigrationName(
		templateName,
		version,
	)

	migrationsDir, err := config.MigrationsDir(true)
	if err != nil {
		return err
	}

	f, err := os.Create(
		path.Join(migrationsDir, migrationName),
	)
	if err != nil {
		return err
	}

	migration.Execute(f, data)

	return nil
}
