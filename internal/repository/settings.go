package repository

import (
	"encoding/json"

	"homepage/internal/model"
)

func decodeSettings(data string) (model.Settings, error) {
	// 旧数据缺少新增字段时预填默认值，避免卡片被隐藏或加载页主题为空
	settings := model.Settings{
		Cards:    model.CardSettings{Profile: true, Navigation: true, Devices: true, Terminal: true, Clock: true},
		Loader:   model.LoaderSettings{Theme: "dark"},
		Terminal: model.TerminalSettings{FastfetchLoadingMS: 2000},
	}
	err := json.Unmarshal([]byte(data), &settings)
	return settings, err
}

func encodeSettings(settings model.Settings) (string, error) {
	data, err := json.Marshal(settings)
	return string(data), err
}
