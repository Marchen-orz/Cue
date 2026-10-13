package app

import (
	"context"

	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/skill"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// pickSkills 为一道题路由并加载相关技能，返回可追加到 system 提示词的片段。
// 命中时通过 event 通知前端展示技能名；无启用技能时不产生任何模型调用。
func (a *App) pickSkills(ctx context.Context, model, text string, images []string, event string) string {
	candidates := a.skills.Enabled()
	if len(candidates) == 0 {
		return ""
	}
	picked := skill.Select(ctx, a.llmService.GetProvider(), model, text, images, candidates)
	if len(picked) == 0 {
		return ""
	}
	a.EmitEvent(event, skill.Names(picked))
	return skill.Block(picked)
}

// ListSkills 已导入的技能列表（不含正文）
func (a *App) ListSkills() []skill.Skill {
	return a.skills.List()
}

// GetSkill 读取单个技能（含正文，用于预览）
func (a *App) GetSkill(id string) (skill.Skill, error) {
	return a.skills.Get(id)
}

// ImportSkillFile 选择 SKILL.md / .zip 导入，取消选择返回空结果
func (a *App) ImportSkillFile() (skill.Skill, error) {
	p, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "导入技能 (SKILL.md / zip)",
		Filters: []runtime.FileFilter{
			{DisplayName: "Skill (*.md, *.zip)", Pattern: "*.md;*.zip"},
		},
	})
	if err != nil || p == "" {
		return skill.Skill{}, err
	}
	sk, err := a.skills.ImportFile(p)
	if err != nil {
		logger.Printf("[skill] 导入失败: %v", err)
		return skill.Skill{}, err
	}
	logger.Printf("[skill] 已导入: %s", sk.ID)
	return sk, nil
}

// ImportSkillDir 选择技能文件夹导入，取消选择返回空结果
func (a *App) ImportSkillDir() (skill.Skill, error) {
	p, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择技能文件夹（含 SKILL.md）",
	})
	if err != nil || p == "" {
		return skill.Skill{}, err
	}
	sk, err := a.skills.ImportDir(p)
	if err != nil {
		logger.Printf("[skill] 导入失败: %v", err)
		return skill.Skill{}, err
	}
	logger.Printf("[skill] 已导入: %s", sk.ID)
	return sk, nil
}

// DeleteSkill 删除技能
func (a *App) DeleteSkill(id string) error {
	return a.skills.Delete(id)
}

// SetSkillEnabled 启用/停用技能
func (a *App) SetSkillEnabled(id string, enabled bool) error {
	return a.skills.SetEnabled(id, enabled)
}
