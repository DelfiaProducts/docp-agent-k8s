package pkg

import (
	"encoding/json"
)

// JsonClient is struct for json functions
type JsonClient struct{}

// NewJsonClient return instance of json client
func NewJsonClient() *JsonClient {
	return &JsonClient{}
}

// Unmarshall execute parse the json for data
func (j *JsonClient) Unmarshall(data []byte, inner any) error {
	if err := json.Unmarshal(data, inner); err != nil {
		return err
	}
	return nil
}

// Marshall execute parse the data for json
func (j *JsonClient) Marshall(inner any) ([]byte, error) {
	data, err := json.Marshal(inner)
	if err != nil {
		return nil, err
	}
	return data, nil
}
