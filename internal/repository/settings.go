package repository

import (
	"encoding/json"

	"homepage/internal/model"
)

func decodeSettings(data string) (model.Settings, error) {
	var settings model.Settings
	err := json.Unmarshal([]byte(data), &settings)
	return settings, err
}

func encodeSettings(settings model.Settings) (string, error) {
	data, err := json.Marshal(settings)
	return string(data), err
}
