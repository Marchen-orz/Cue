// Package modelcaps 拉取并缓存最新的模型能力表（供前端展示图标/参数兜底）。
// 数据源为 OpenRouter 公开模型列表，格式与前端内置 model-capabilities.json 一致。
package modelcaps

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Q-Solver/pkg/common"
	"Q-Solver/pkg/logger"
)

const openRouterModelsURL = "https://openrouter.ai/api/v1/models"

// cacheFileName 能力表缓存文件名（位于 %AppData%/Cue/）
const cacheFileName = "model-cache.json"

// entry 与前端 model-capabilities.json 的单条目结构一致
type entry struct {
	Provider      string   `json:"provider"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	SupportsImage bool     `json:"supports_image,omitempty"`
	ContextLength int      `json:"context_length"`
	Inputs        []string `json:"inputs,omitempty"`
	Outputs       []string `json:"outputs,omitempty"`
}

// CachePath 返回缓存文件路径
func CachePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, common.AppName, cacheFileName), nil
}

// openRouterResponse OpenRouter /api/v1/models 返回结构（仅取所需字段）
type openRouterResponse struct {
	Data []struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Description   string `json:"description"`
		ContextLength int    `json:"context_length"`
		Architecture  struct {
			InputModalities  []string `json:"input_modalities"`
			OutputModalities []string `json:"output_modalities"`
		} `json:"architecture"`
	} `json:"data"`
}

// SyncFromOpenRouter 非阻塞地从 OpenRouter 拉取模型元数据，转换为前端能力表格式后写入本地缓存。
// 供启动时 SafeGo 调用；网络失败时静默保留旧缓存，不影响启动流程。
func SyncFromOpenRouter() {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(openRouterModelsURL)
	if err != nil {
		logger.Printf("[modelcaps] 拉取模型列表失败（保留旧缓存）: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Printf("[modelcaps] 拉取模型列表返回 %d（保留旧缓存）", resp.StatusCode)
		return
	}

	var parsed openRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		logger.Printf("[modelcaps] 解析模型列表失败: %v", err)
		return
	}

	models := make(map[string]entry, len(parsed.Data))
	for _, m := range parsed.Data {
		parts := strings.SplitN(m.ID, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			continue
		}
		inputs := m.Architecture.InputModalities
		hasImage := false
		for _, in := range inputs {
			if in == "image" {
				hasImage = true
				break
			}
		}
		models[strings.ToLower(parts[1])] = entry{
			Provider:      parts[0],
			Name:          m.Name,
			Description:   m.Description,
			SupportsImage: hasImage,
			ContextLength: m.ContextLength,
			Inputs:        inputs,
			Outputs:       m.Architecture.OutputModalities,
		}
	}

	if len(models) == 0 {
		logger.Println("[modelcaps] 模型列表为空（保留旧缓存）")
		return
	}

	data, err := json.Marshal(models)
	if err != nil {
		logger.Printf("[modelcaps] 序列化缓存失败: %v", err)
		return
	}

	path, err := CachePath()
	if err != nil {
		logger.Printf("[modelcaps] 获取缓存路径失败: %v", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		logger.Printf("[modelcaps] 创建缓存目录失败: %v", err)
		return
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		logger.Printf("[modelcaps] 写入缓存失败: %v", err)
		return
	}
	logger.Printf("[modelcaps] 模型能力表已更新: %d 个模型 -> %s", len(models), path)
}

// LoadCache 读取本地缓存内容；无缓存或读取失败返回空串
func LoadCache() string {
	path, err := CachePath()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
