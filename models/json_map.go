package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

var ErrModelsJSONmapUnknownType = errors.New("models jsonmap unknown type")

type JSONmap map[string]any

func (jm JSONmap) Value() (driver.Value, error) {
	if jm == nil {
		return nil, nil
	}
	return json.Marshal(jm)
}

func (jm *JSONmap) Scan(value any) error {
	if value == nil {
		*jm = nil
		return nil
	}

	data := []byte{}
	switch v := value.(type) {
	case []byte:
		data = v
	default:
		return ErrModelsJSONmapUnknownType
	}

	return json.Unmarshal(data, jm)
}
