package helper

import "github.com/bytedance/sonic"

func Marshal(object interface{}) ([]byte, error) {
	var output []byte
	output, err := sonic.Marshal(object)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func Unmarshal[S any](json []byte) (*S, error) {
	var s = new(S)
	err := sonic.Unmarshal(json, s)
	return s, err
}

func Convertor[S any](source interface{}) (*S, error) {
	if source == nil {
		return nil, nil
	}
	json, err := Marshal(source)
	if err != nil {
		return nil, err
	}

	des, err := Unmarshal[S](json)
	if err != nil {
		return nil, err
	}

	return des, err
}
