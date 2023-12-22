package helper

import (
	"encoding/csv"
	"os"
)

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
