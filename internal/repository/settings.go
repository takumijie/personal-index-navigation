package repository

import (
	"encoding/json"

	"homepage/internal/model"
)

func decodeSettings(data string) (model.Settings, error) {
	// 旧数据没有 cards 字段，预填全开避免 bool 零值隐藏所有卡片
	settings := model.Settings{
		Cards: model.CardSettings{Profile: true, Navigation: true, Devices: true, Terminal: true, Clock: true},
	}
	err := json.Unmarshal([]byte(data), &settings)
	return settings, err
}

func encodeSettings(settings model.Settings) (string, error) {
	data, err := json.Marshal(settings)
	return string(data), err
}
