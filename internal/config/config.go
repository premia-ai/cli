package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
)

func getDir(dirPath string, createIfMissing bool) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := path.Join(homeDir, dirPath)

	_, err = os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.Mkdir(dir, 0777)
		if !createIfMissing {
			return "", errors.New(
				fmt.Sprintf("'%s' directory doesn't exist.", dirPath),
			)
		}

		if err != nil {
			return "", err
		}
	}

	return dir, err
}

func ConfigDir(createIfMissing bool) (string, error) {
	return getDir(".premia", createIfMissing)
}

func MigrationsDir(createIfMissing bool) (string, error) {
	return getDir(".premia/migrations", createIfMissing)
}

func TmpDir(createIfMissing bool) (string, error) {
	return getDir(".premia/tmp", createIfMissing)
}

type ConfigFileData struct {
	Version      string `json:"version"`
	BaseTable    string `json:"baseTable,omitempty"`
	TimespanUnit string `json:"timespan,omitempty"`
}

func CreateConfigFileData(baseTable, timespanUnit string) *ConfigFileData {
	return &ConfigFileData{
		Version:      "1",
		BaseTable:    baseTable,
		TimespanUnit: timespanUnit,
	}
}

func UpdateConfigFile(data *ConfigFileData) error {
	configFile, err := configFile()
	if err != nil {
		return err
	}
	defer configFile.Close()

	err = configFile.Truncate(0)
	if err != nil {
		return err
	}
	_, err = configFile.Seek(0, 0)
	if err != nil {
		return err
	}

	fileContent, err := jsonPrettyPrint(data)
	if err != nil {
		return err
	}

	_, err = configFile.Write(fileContent)
	if err != nil {
		return err
	}

	return nil
}

func Config() (*ConfigFileData, error) {
	configDir, err := ConfigDir(false)
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(path.Join(configDir, "config.json"))
	if err != nil {
		return nil, err
	}

	var data *ConfigFileData
	err = json.Unmarshal(content, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func configFile() (*os.File, error) {
	configDir, err := ConfigDir(true)
	if err != nil {
		return nil, err
	}

	configFilePath := path.Join(configDir, "config.json")
	_, err = os.Stat(configFilePath)

	var f *os.File
	if os.IsNotExist(err) {
		f, err = os.Create(configFilePath)
		if err != nil {
			return nil, err
		}

		fileContent, err := jsonPrettyPrint(CreateConfigFileData("", ""))
		if err != nil {
			return nil, err
		}

		_, err = f.Write(fileContent)
		if err != nil {
			return nil, err
		}
	} else {
		f, err = os.OpenFile(configFilePath, os.O_RDWR, 0666)
		if err != nil {
			return nil, err
		}
	}

	return f, nil
}

func SetupConfigDir() (string, error) {
	configDir, err := ConfigDir(true)
	if err != nil {
		return "", err
	}

	f, err := configFile()
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = MigrationsDir(true)
	if err != nil {
		return "", err
	}

	_, err = TmpDir(true)
	if err != nil {
		return "", err
	}

	return configDir, nil
}

func jsonPrettyPrint(value any) ([]byte, error) {
	result, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}

	result = append(result, byte('\n'))
	return result, nil
}
