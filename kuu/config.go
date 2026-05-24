package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mgenware/j9/v3"
	"github.com/mgenware/ku-builder"
)

func ReadKuConfig(shell *ku.Shell) map[string]interface{} {
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

	shell.Log(j9.LogLevelInfo, "Read .ku.json successfully")
	return result
}

func ReadConfigString(config map[string]interface{}, key string) string {
	if value, ok := config[key]; ok {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return ""
}

func ReadConfigStringArray(config map[string]interface{}, key string) []string {
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

func ReadConfigMap(config map[string]interface{}, key string) map[string]interface{} {
	if value, ok := config[key]; ok {
		if mapValue, ok := value.(map[string]interface{}); ok {
			return mapValue
		}
	}
	return nil
}
