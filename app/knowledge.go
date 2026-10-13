package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"Q-Solver/pkg/config"
	"Q-Solver/pkg/knowledge"
	"Q-Solver/pkg/logger"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// KnowledgeImportResult 一次导入的结果：成功的文档（不含正文）与逐个失败原因
type KnowledgeImportResult struct {
	Imported []knowledge.Doc `json:"imported"`
	Failed   []string        `json:"failed"`
}

// GetKnowledge 知识库概览（文档不含正文、已用/上限字数）
func (a *App) GetKnowledge() knowledge.Summary {
	return a.knowledge.List()
}

// GetKnowledgeDoc 读取单个文档（含正文）
func (a *App) GetKnowledgeDoc(id string) (knowledge.Doc, error) {
	return a.knowledge.Get(id)
}

// SaveKnowledgeDoc 保存文档：id 为空 = 新建笔记，否则修改该文档的名称与正文
func (a *App) SaveKnowledgeDoc(id, name, content string) (knowledge.Doc, error) {
	if id == "" {
		return a.knowledge.Add(name, knowledge.KindNote, content)
	}
	return a.knowledge.Update(id, name, content)
}

// SetKnowledgeScope 设置文档的使用范围：interview（面试）/ solve（截图搜题）/ both（两者）
func (a *App) SetKnowledgeScope(id, scope string) error {
	return a.knowledge.SetScope(id, scope)
}

// DeleteKnowledgeDoc 删除文档
func (a *App) DeleteKnowledgeDoc(id string) error {
	return a.knowledge.Delete(id)
}

// ImportKnowledgeResume 选择简历文件导入知识库（PDF 由模型整理为结构化 Markdown；
// 知识库中至多一份简历，再次导入会替换）。取消选择返回空结果。
func (a *App) ImportKnowledgeResume() (KnowledgeImportResult, error) {
	p, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择简历 (PDF / DOCX / MD / TXT)",
		Filters: []runtime.FileFilter{
			{DisplayName: "简历 (*.pdf, *.docx, *.md, *.txt)", Pattern: "*.pdf;*.docx;*.md;*.markdown;*.txt"},
		},
	})
	if err != nil || p == "" {
		return KnowledgeImportResult{}, err
	}
	return a.importKnowledgePaths([]string{p}, true), nil
}

// ImportKnowledgeFiles 选择一个或多个文档导入知识库（PDF 由模型转写为 Markdown）。
// 取消选择返回空结果。
func (a *App) ImportKnowledgeFiles() (KnowledgeImportResult, error) {
	paths, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "导入文档 (PDF / DOCX / MD / TXT)",
		Filters: []runtime.FileFilter{
			{DisplayName: "文档 (*.pdf, *.docx, *.md, *.txt)", Pattern: "*.pdf;*.docx;*.md;*.markdown;*.txt"},
		},
	})
	if err != nil || len(paths) == 0 {
		return KnowledgeImportResult{}, err
	}
	return a.importKnowledgePaths(paths, false), nil
}

func (a *App) importKnowledgePaths(paths []string, asResume bool) KnowledgeImportResult {
	res := KnowledgeImportResult{Imported: []knowledge.Doc{}, Failed: []string{}}
	for _, p := range paths {
		base := filepath.Base(p)
		doc, err := a.importKnowledgeFile(p, asResume)
		if err != nil {
			logger.Printf("[knowledge] 导入失败 %s: %v", base, err)
			res.Failed = append(res.Failed, fmt.Sprintf("%s：%v", base, err))
			continue
		}
		logger.Printf("[knowledge] 已导入 %s（%d 字）", doc.Name, doc.Size)
		res.Imported = append(res.Imported, doc)
	}
	return res
}

func (a *App) importKnowledgeFile(path string, asResume bool) (knowledge.Doc, error) {
	// PDF 要调用模型，先快速判断额度，避免白白消耗一次请求
	if !asResume && a.knowledge.Remaining() <= 0 {
		return knowledge.Doc{}, fmt.Errorf("知识库已满（上限 %d 字），请先删减其它文档", knowledge.MaxTotalRunes)
	}

	var (
		content string
		err     error
	)
	if knowledge.IsPDF(path) {
		content, err = knowledge.ParsePDF(a.ctx, a.configManager.Get(), path, asResume)
	} else {
		content, err = knowledge.ExtractLocal(path)
	}
	if err != nil {
		return knowledge.Doc{}, err
	}

	kind := knowledge.KindDoc
	if asResume {
		kind = knowledge.KindResume
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return a.knowledge.Add(name, kind, content)
}

// migrateLegacyResume 把旧版存放在配置里的简历（ResumeContent）一次性迁入知识库。
// 迁移成功后清空旧字段；失败（如超出字数上限）时保留旧数据，下次启动再试。
func (a *App) migrateLegacyResume() {
	cfg := a.configManager.Get()
	if strings.TrimSpace(cfg.ResumeContent) == "" {
		if cfg.ResumePath != "" {
			_ = a.configManager.Patch(func(c *config.Config) { c.ResumePath = "" })
		}
		return
	}

	name := "简历"
	if cfg.ResumePath != "" {
		name = strings.TrimSuffix(filepath.Base(cfg.ResumePath), filepath.Ext(cfg.ResumePath))
	}
	if _, err := a.knowledge.Add(name, knowledge.KindResume, cfg.ResumeContent); err != nil {
		logger.Printf("[knowledge] 旧简历迁移失败（保留原数据）: %v", err)
		return
	}
	_ = a.configManager.Patch(func(c *config.Config) {
		c.ResumePath = ""
		c.ResumeContent = ""
	})
	logger.Println("[knowledge] 旧简历已迁入知识库")
}
