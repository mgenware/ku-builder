package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder"
)

var config map[string]interface{}

func init() {
	config = make(map[string]interface{})
}

func InitKuConfig(shell *ku.Shell) {
	// Read the file
	data, err := os.ReadFile(".ku.json")
	if err != nil {
		shell.Quit(fmt.Sprintf("Error reading .ku.json: %v\n", err))
	}

	// Parse JSON into map
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		shell.Quit(fmt.Sprintf("Error parsing .ku.json: %v\n", err))
	}

	config = result
	shell.Logger().Log(j9.LogLevelInfo, "✅ Read .ku.json successfully\n")
	for key, value := range config {
		shell.Logger().Log(j9.LogLevelVerbose, fmt.Sprintf("%s: %v\n", key, value))
	}
}

func ReadKuConfigString(key string) string {
	if value, ok := config[key]; ok {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return ""
}

func ReadKuConfigStringArray(key string) []string {
	if value, ok := config[key]; ok {
		if arrValue, ok := value.([]interface{}); ok {
			strArr := make([]string, len(arrValue))
			for i, v := range arrValue {
				if strV, ok := v.(string); ok {
					strArr[i] = strV
				}
			}
			return strArr
		}
	}
	return nil
}
