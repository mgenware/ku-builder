package main

import (
	"encoding/json"
	"fmt"
	"os"
)

var config map[string]interface{}

func init() {
	config = make(map[string]interface{})
}

func InitKuConfig() {
	// Read the file
	data, err := os.ReadFile(".ku.json")
	if err != nil {
		fmt.Printf("Error reading .ku.json: %v\n", err)
		return
	}

	// Parse JSON into map
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		fmt.Printf("Error parsing .ku.json: %v\n", err)
		return
	}

	config = result
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
