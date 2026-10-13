package knowledge

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/prompts"
)

// ParsePDF 通过当前配置的模型把 PDF 解析为 Markdown。
// asResume 为 true 时按简历结构重新整理，否则按原文忠实转写。
// 需要模型支持 PDF / 文件输入。
func ParsePDF(ctx context.Context, cfg config.Config, path string, asResume bool) (string, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return "", errors.New("请先配置 API Key")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return "", errors.New("请先选择模型")
	}

	data, err := readLimited(path)
	if err != nil {
		return "", err
	}

	system, ask := prompts.DocumentParsePrompt, "请将这份文档完整转写为 Markdown。"
	if asResume {
		system, ask = prompts.ResumeParsePrompt, "请将这份简历解析并整理为结构清晰的 Markdown。"
	}

	logger.Printf("[knowledge] 通过模型解析 PDF（简历=%v，%d 字节）", asResume, len(data))
	adapter := llm.NewOpenAIAdapter(&cfg)
	result, err := adapter.GenerateContent(ctx, cfg.Model, []llm.Message{
		llm.NewSystemMessage(system),
		llm.NewMultiPartMessage(llm.RoleUser, []llm.ContentPart{
			llm.TextPart(ask),
			llm.PDFPart(base64.StdEncoding.EncodeToString(data)),
		}),
	})
	if err != nil {
		logger.Printf("[knowledge] PDF 解析失败: %v", err)
		return "", fmt.Errorf("模型解析 PDF 失败（需要模型支持 PDF / 文件输入，也可先把内容复制为文本后导入）: %w", err)
	}

	content := strings.TrimSpace(result.Content)
	if content == "" {
		return "", errors.New("模型没有返回解析结果")
	}
	return content, nil
}
