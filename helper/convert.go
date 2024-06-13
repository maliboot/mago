package helper

import (
	"encoding/json"
	"github.com/iancoleman/strcase"
)

func Marshal(object interface{}) ([]byte, error) {
	var output []byte
	output, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func Unmarshal[S any](jsonData []byte) (*S, error) {
	var s = new(S)
	err := json.Unmarshal(jsonData, s)
	return s, err
}

func ToLowerCamelJson(snakeJson []byte) ([]byte, error) {
	var data interface{}
	err := json.Unmarshal(snakeJson, &data)
	if err != nil {
		return nil, err
	}

	result := convertKeysCase(data, false)
	return json.Marshal(result)
}

func MapFormatCase(data map[string]interface{}, toSnake bool) map[string]interface{} {
	result := convertKeysCase(data, toSnake)
	return result.(map[string]interface{})
}

func ArrayMapFormatCase(data []map[string]interface{}, toSnake bool) []map[string]interface{} {
	result := make([]map[string]interface{}, len(data))

	for i, datum := range data {
		result[i] = convertKeysCase(datum, toSnake).(map[string]interface{})
	}
	return result
}

func ToSnakeJson(lowerCamelJson []byte) ([]byte, error) {
	var data interface{}
	err := json.Unmarshal(lowerCamelJson, &data)
	if err != nil {
		return nil, err
	}

	result := convertKeysCase(data, true)
	return json.Marshal(result)
}

func convertKeysCase(input interface{}, isSnake bool) interface{} {
	switch value := input.(type) {
	case map[string]interface{}:
		outputMap := make(map[string]interface{})
		for key, val := range value {
			var caseKey string
			if isSnake {
				caseKey = strcase.ToSnake(key)
			} else {
				caseKey = strcase.ToLowerCamel(key)
			}
			outputMap[caseKey] = convertKeysCase(val, isSnake)
		}
		return outputMap
	case []interface{}:
		for i, item := range value {
			value[i] = convertKeysCase(item, isSnake)
		}
		return value
	default:
		return input
	}
}

func Convertor[S any](source interface{}) (*S, error) {
	if source == nil {
		return nil, nil
	}
	jsonData, err := Marshal(source)
	if err != nil {
		return nil, err
	}
	return Unmarshal[S](jsonData)
}

func LowerCamelConvertor[S any](source interface{}) (*S, error) {
	if source == nil {
		return nil, nil
	}
	jsonData, err := Marshal(source)
	if err != nil {
		return nil, err
	}
	if _, ok := source.(json.Marshaler); ok {
		return Unmarshal[S](jsonData)
	}

	camelJson, err := ToLowerCamelJson(jsonData)
	if err != nil {
		return nil, err
	}
	return Unmarshal[S](camelJson)
}

// SnakeConvertor 默认转字段key为蛇形
func SnakeConvertor[S any](source interface{}) (*S, error) {
	if source == nil {
		return nil, nil
	}
	jsonData, err := Marshal(source)
	if err != nil {
		return nil, err
	}
	if _, ok := source.(json.Marshaler); ok {
		return Unmarshal[S](jsonData)
	}

	camelJson, err := ToSnakeJson(jsonData)
	if err != nil {
		return nil, err
	}
	return Unmarshal[S](camelJson)
}
