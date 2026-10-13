// Package skill 技能（Skill）管理：导入 Claude SKILL.md 格式的技能，
// 答题时由路由模型按题目挑选相关技能并注入系统提示词。
//
// 存储布局（<UserConfigDir>/Cue/skills）：
//
//	<id>/SKILL.md   技能原文（frontmatter + Markdown 正文）
//	state.json      {"disabled": ["<id>", ...]}  被停用的技能
package skill

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"Q-Solver/pkg/common"
)

const (
	skillFileName = "SKILL.md"
	stateFileName = "state.json"

	maxSkillBytes = 256 * 1024 // 单个 SKILL.md 大小上限
	maxScanDepth  = 3          // 导入文件夹时查找 SKILL.md 的最大深度
)

// Skill 一个已导入的技能
type Skill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Body        string `json:"body,omitempty"` // 仅 Get 返回；List 不带正文
	Size        int    `json:"size"`           // 正文字符数
	Enabled     bool   `json:"enabled"`
}

// Store 技能存储
type Store struct {
	mu  sync.Mutex
	dir string
}

// NewStore 创建存储（目录不存在时自动创建）
func NewStore() *Store {
	base, err := os.UserConfigDir()
	if err != nil {
		base = "."
	}
	dir := filepath.Join(base, common.AppName, "skills")
	_ = os.MkdirAll(dir, 0755)
	return &Store{dir: dir}
}

// ==================== 查询 ====================

// List 返回全部技能（不含正文），按名称排序
func (s *Store) List() []Skill {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listLocked(false)
}

// Enabled 返回已启用的技能（含正文，供路由与注入使用）
func (s *Store) Enabled() []Skill {
	s.mu.Lock()
	defer s.mu.Unlock()
	all := s.listLocked(true)
	out := all[:0]
	for _, sk := range all {
		if sk.Enabled {
			out = append(out, sk)
		}
	}
	return out
}

// Get 读取单个技能（含正文）
func (s *Store) Get(id string) (Skill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked(id, s.disabledLocked())
}

func (s *Store) listLocked(withBody bool) []Skill {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil
	}
	disabled := s.disabledLocked()
	list := make([]Skill, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sk, err := s.loadLocked(e.Name(), disabled)
		if err != nil {
			continue
		}
		if !withBody {
			sk.Body = ""
		}
		list = append(list, sk)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list
}

func (s *Store) loadLocked(id string, disabled map[string]bool) (Skill, error) {
	if !validID(id) {
		return Skill{}, fmt.Errorf("无效的技能 ID")
	}
	data, err := os.ReadFile(filepath.Join(s.dir, id, skillFileName))
	if err != nil {
		return Skill{}, err
	}
	name, desc, body := ParseSkillMD(string(data), id)
	return Skill{
		ID:          id,
		Name:        name,
		Description: desc,
		Body:        body,
		Size:        len([]rune(body)),
		Enabled:     !disabled[id],
	}, nil
}

// ==================== 变更 ====================

// ImportFile 导入单个 .md 文件或 .zip 压缩包（包内需含 SKILL.md）
func (s *Store) ImportFile(p string) (Skill, error) {
	var content []byte
	var err error
	switch strings.ToLower(filepath.Ext(p)) {
	case ".zip":
		content, err = readSkillFromZip(p)
	case ".md", ".markdown", ".txt":
		content, err = readLimited(p)
	default:
		return Skill{}, fmt.Errorf("不支持的文件类型，请选择 SKILL.md 或 .zip")
	}
	if err != nil {
		return Skill{}, err
	}
	fallback := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
	if strings.EqualFold(fallback, "skill") {
		fallback = filepath.Base(filepath.Dir(p))
	}
	return s.save(content, fallback)
}

// ImportDir 导入技能文件夹（文件夹内或其子目录中需含 SKILL.md）
func (s *Store) ImportDir(dir string) (Skill, error) {
	found, err := findSkillFile(dir, 0)
	if err != nil {
		return Skill{}, err
	}
	content, err := readLimited(found)
	if err != nil {
		return Skill{}, err
	}
	return s.save(content, filepath.Base(filepath.Dir(found)))
}

// Delete 删除技能
func (s *Store) Delete(id string) error {
	if !validID(id) {
		return fmt.Errorf("无效的技能 ID")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.RemoveAll(filepath.Join(s.dir, id)); err != nil {
		return err
	}
	disabled := s.disabledLocked()
	if disabled[id] {
		delete(disabled, id)
		return s.writeDisabledLocked(disabled)
	}
	return nil
}

// SetEnabled 启用/停用技能
func (s *Store) SetEnabled(id string, enabled bool) error {
	if !validID(id) {
		return fmt.Errorf("无效的技能 ID")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	disabled := s.disabledLocked()
	if enabled {
		delete(disabled, id)
	} else {
		disabled[id] = true
	}
	return s.writeDisabledLocked(disabled)
}

// save 校验并落盘。同名技能（ID 相同）覆盖更新，保留启用状态。
func (s *Store) save(content []byte, fallbackName string) (Skill, error) {
	text := strings.TrimPrefix(string(content), "\uFEFF")
	if strings.TrimSpace(text) == "" {
		return Skill{}, errors.New("技能文件为空")
	}
	name, _, body := ParseSkillMD(text, fallbackName)
	if strings.TrimSpace(body) == "" {
		return Skill{}, errors.New("技能正文为空")
	}
	id := slugify(name)
	if id == "" {
		id = slugify(fallbackName)
	}
	if id == "" {
		id = "skill"
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(s.dir, id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return Skill{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, skillFileName), []byte(text), 0644); err != nil {
		return Skill{}, err
	}
	return s.loadLocked(id, s.disabledLocked())
}

// ==================== 启用状态 ====================

func (s *Store) disabledLocked() map[string]bool {
	out := map[string]bool{}
	data, err := os.ReadFile(filepath.Join(s.dir, stateFileName))
	if err != nil {
		return out
	}
	var st struct {
		Disabled []string `json:"disabled"`
	}
	if json.Unmarshal(data, &st) == nil {
		for _, id := range st.Disabled {
			out[id] = true
		}
	}
	return out
}

func (s *Store) writeDisabledLocked(disabled map[string]bool) error {
	ids := make([]string, 0, len(disabled))
	for id := range disabled {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	data, _ := json.Marshal(struct {
		Disabled []string `json:"disabled"`
	}{ids})
	return os.WriteFile(filepath.Join(s.dir, stateFileName), data, 0644)
}

// ==================== 解析 ====================

// ParseSkillMD 解析 SKILL.md：frontmatter 的 name / description + 正文。
// 缺失时 name 回退为 fallbackName，description 回退为正文首个非标题行。
// frontmatter 仅支持 Claude skill 常见写法：单行值、引号值、`>` / `|` 多行块。
func ParseSkillMD(content, fallbackName string) (name, description, body string) {
	content = strings.ReplaceAll(strings.TrimPrefix(content, "\uFEFF"), "\r\n", "\n")
	body = content

	if strings.HasPrefix(content, "---\n") {
		if end := strings.Index(content[4:], "\n---"); end >= 0 {
			fm := content[4 : 4+end]
			body = strings.TrimPrefix(content[4+end+4:], "\n")
			name, description = parseFrontmatter(fm)
		}
	}
	body = strings.TrimSpace(body)
	name = strings.TrimSpace(name)
	if name == "" {
		name = strings.TrimSpace(fallbackName)
	}
	if name == "" {
		name = "未命名技能"
	}
	if description == "" {
		for _, line := range strings.Split(body, "\n") {
			line = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "#"))
			if line != "" {
				description = line
				break
			}
		}
	}
	description = truncateRunes(strings.TrimSpace(description), 400)
	return
}

func parseFrontmatter(fm string) (name, description string) {
	lines := strings.Split(fm, "\n")
	for i := 0; i < len(lines); i++ {
		key, val, ok := strings.Cut(lines[i], ":")
		if !ok || strings.HasPrefix(lines[i], " ") || strings.HasPrefix(lines[i], "\t") {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key != "name" && key != "description" {
			continue
		}
		if val == ">" || val == "|" || val == ">-" || val == "|-" {
			// 多行块：收集后续缩进行
			var parts []string
			for i+1 < len(lines) && (strings.HasPrefix(lines[i+1], " ") || strings.HasPrefix(lines[i+1], "\t") || strings.TrimSpace(lines[i+1]) == "") {
				i++
				if t := strings.TrimSpace(lines[i]); t != "" {
					parts = append(parts, t)
				}
			}
			val = strings.Join(parts, " ")
		} else {
			val = strings.Trim(val, `"'`)
		}
		if key == "name" {
			name = val
		} else {
			description = val
		}
	}
	return
}

// ==================== 工具 ====================

var (
	slugRe = regexp.MustCompile(`[^\p{L}\p{N}]+`)
	idRe   = regexp.MustCompile(`^[\p{L}\p{N}][\p{L}\p{N}_-]*$`)
)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Trim(slugRe.ReplaceAllString(s, "-"), "-")
	return truncateRunes(s, 60)
}

// validID 防止路径穿越：ID 只允许字母数字与 -_
func validID(id string) bool {
	return id != "" && idRe.MatchString(id)
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func readLimited(p string) ([]byte, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxSkillBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxSkillBytes {
		return nil, fmt.Errorf("技能文件超过 %dKB 限制", maxSkillBytes/1024)
	}
	return data, nil
}

// findSkillFile 在目录中查找 SKILL.md（广度优先，限制深度）
func findSkillFile(dir string, depth int) (string, error) {
	direct := filepath.Join(dir, skillFileName)
	if st, err := os.Stat(direct); err == nil && !st.IsDir() {
		return direct, nil
	}
	if depth >= maxScanDepth {
		return "", fmt.Errorf("未找到 %s", skillFileName)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") && e.Name() != "node_modules" {
			if p, err := findSkillFile(filepath.Join(dir, e.Name()), depth+1); err == nil {
				return p, nil
			}
		}
	}
	return "", fmt.Errorf("所选文件夹中未找到 %s", skillFileName)
}

// readSkillFromZip 读取压缩包中层级最浅的 SKILL.md
func readSkillFromZip(p string) ([]byte, error) {
	zr, err := zip.OpenReader(p)
	if err != nil {
		return nil, fmt.Errorf("无法打开压缩包: %w", err)
	}
	defer zr.Close()

	var best *zip.File
	bestDepth := 1 << 30
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || !strings.EqualFold(path.Base(f.Name), skillFileName) {
			continue
		}
		if d := strings.Count(strings.Trim(f.Name, "/"), "/"); d < bestDepth {
			best, bestDepth = f, d
		}
	}
	if best == nil {
		return nil, fmt.Errorf("压缩包中未找到 %s", skillFileName)
	}
	if best.UncompressedSize64 > maxSkillBytes {
		return nil, fmt.Errorf("技能文件超过 %dKB 限制", maxSkillBytes/1024)
	}
	rc, err := best.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, maxSkillBytes+1))
}
