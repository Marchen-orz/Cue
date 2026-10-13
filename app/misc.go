package app

import (
	"Q-Solver/pkg/modelcaps"
	"encoding/base64"
	"os"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) GetInitStatus() string {
	return a.stateManager.GetInitStatusString()
}

// GetModelCapabilitiesCache 返回本地缓存的最新模型能力表（JSON 字符串；无缓存返回空串）。
// 前端启动时异步读取，与内置 capabilities 合并，用于展示内置表未收录的模型图标/参数。
func (a *App) GetModelCapabilitiesCache() string {
	return modelcaps.LoadCache()
}

func (a *App) SaveImageToFile(base64Data string) (bool, error) {
	filename, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "保存图片",
		DefaultFilename: "cue-export.png",
		Filters: []runtime.FileFilter{
			{DisplayName: "PNG 图片", Pattern: "*.png"},
		},
	})
	if err != nil {
		return false, err
	}
	if filename == "" {
		return false, nil
	}

	const prefix = "data:image/png;base64,"
	data := base64Data
	if len(data) > len(prefix) && data[:len(prefix)] == prefix {
		data = data[len(prefix):]
	}

	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return false, err
	}

	if err := os.WriteFile(filename, decoded, 0644); err != nil {
		return false, err
	}

	return true, nil
}
