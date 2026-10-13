// Package knowledge 知识库：面试问答的背景资料（简历、项目与技术笔记等）。
//
// 为了回答速度，知识库不做检索，而是把所有文档全量附加到面试的系统提示词中，
// 因此对全部文档的总字数设有上限（MaxTotalRunes）。
// 每份文档可单独选择用于面试、截图搜题或两者（Doc.Scope），默认只用于面试。
//
// 存储（<UserConfigDir>/Cue/knowledge）：所有文档序列化为一个 JSON，
// 用与配置文件相同的 AES-GCM 加密后整体写入。
package knowledge

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"Q-Solver/pkg/common"
	"Q-Solver/pkg/config"
	"Q-Solver/pkg/logger"
)

const (
	// MaxTotalRunes 知识库全部文档的总字数上限（按字符数计）。
	// 全量注入提示词，过长会拖慢首字延迟，所以限制得比较克制。
	MaxTotalRunes = 20000

	// 文档类型
	KindResume = "resume" // 简历（至多一份，再次导入会替换）
	KindDoc    = "doc"    // 导入的文档
	KindNote   = "note"   // 手动新建的笔记

	// 文档的使用范围
	ScopeInterview = "interview" // 仅面试模式（默认）
	ScopeSolve     = "solve"     // 仅截图搜题
	ScopeBoth      = "both"      // 面试与搜题都使用

	maxNameRunes = 60
	fileName     = "knowledge"
)

// Doc 知识库中的一份文档
type Doc struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Scope     string `json:"scope"`             // ScopeInterview / ScopeSolve / ScopeBoth
	Content   string `json:"content,omitempty"` // 仅 Get 返回；List 不带正文
	Size      int    `json:"size"`              // 字数
	UpdatedAt int64  `json:"updatedAt"`         // Unix 秒
}

// Summary 知识库概览（不含正文）
type Summary struct {
	Docs  []Doc `json:"docs"`
	Used  int   `json:"used"`  // 已用字数
	Limit int   `json:"limit"` // 字数上限
}

// Store 知识库存储
type Store struct {
	mu   sync.Mutex
	path string
	docs []Doc
}

// NewStore 创建并加载知识库（文件不存在时为空）
func NewStore() *Store {
	base, err := os.UserConfigDir()
	if err != nil {
		base = "."
	}
	dir := filepath.Join(base, common.AppName)
	_ = os.MkdirAll(dir, 0755)
	s := &Store{path: filepath.Join(dir, fileName)}
	s.load()
	return s
}

// ==================== 查询 ====================

// List 返回知识库概览（文档不含正文）。简历排在最前，其余按添加顺序。
func (s *Store) List() Summary {
	s.mu.Lock()
	defer s.mu.Unlock()
	docs := make([]Doc, 0, len(s.docs))
	for _, d := range s.sortedLocked() {
		d.Content = ""
		docs = append(docs, d)
	}
	return Summary{Docs: docs, Used: s.usedLocked(-1), Limit: MaxTotalRunes}
}

// Get 读取单个文档（含正文）
func (s *Store) Get(id string) (Doc, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.indexLocked(id)
	if i < 0 {
		return Doc{}, errors.New("文档不存在")
	}
	return s.docs[i], nil
}

// Prompt 渲染为可追加到系统提示词的片段，只包含适用于 use（ScopeInterview / ScopeSolve）的文档；
// 没有适用文档时返回空串。
func (s *Store) Prompt(use string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var docs []Doc
	for _, d := range s.sortedLocked() {
		if d.Scope == ScopeBoth || d.Scope == use {
			docs = append(docs, d)
		}
	}
	if len(docs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n<KnowledgeBase>\n")
	b.WriteString("  <Note>以下是候选人的背景资料（简历、项目与技术笔记等）。涉及候选人的经历、技术栈、项目与成果时以这些资料为准，保持事实一致，资料里没有的经历不要编造；与当前问题无关的内容不必引用。</Note>\n")
	for _, d := range docs {
		fmt.Fprintf(&b, "  <Document name=%q kind=%q>\n%s\n  </Document>\n", d.Name, d.Kind, d.Content)
	}
	b.WriteString("</KnowledgeBase>\n")
	return b.String()
}

// ==================== 变更 ====================

// Add 新增文档。KindResume 至多一份：已有简历时替换之。
// 加入后总字数超过上限时拒绝，返回 *OverLimitError。
func (s *Store) Add(name, kind, content string) (Doc, error) {
	content = Normalize(content)
	if content == "" {
		return Doc{}, errors.New("文档内容为空")
	}
	if kind != KindResume && kind != KindDoc && kind != KindNote {
		kind = KindDoc
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	replace := -1
	if kind == KindResume {
		for i, d := range s.docs {
			if d.Kind == KindResume {
				replace = i
				break
			}
		}
	}
	ignoreID := ""
	if replace >= 0 {
		ignoreID = s.docs[replace].ID
	}
	if err := s.checkQuotaLocked(ignoreID, content); err != nil {
		return Doc{}, err
	}

	doc := Doc{
		Name:      cleanName(name, kind),
		Kind:      kind,
		Scope:     ScopeInterview,
		Content:   content,
		Size:      utf8.RuneCountInString(content),
		UpdatedAt: time.Now().Unix(),
	}
	if replace >= 0 {
		// 替换简历时沿用原文档的 ID 与使用范围
		doc.ID = s.docs[replace].ID
		doc.Scope = s.docs[replace].Scope
		s.docs[replace] = doc
	} else {
		doc.ID = newID()
		s.docs = append(s.docs, doc)
	}
	if err := s.saveLocked(); err != nil {
		return Doc{}, err
	}
	doc.Content = ""
	return doc, nil
}

// Update 修改文档名称与正文。修改后总字数超过上限时拒绝。
func (s *Store) Update(id, name, content string) (Doc, error) {
	content = Normalize(content)
	if content == "" {
		return Doc{}, errors.New("文档内容为空")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.indexLocked(id)
	if i < 0 {
		return Doc{}, errors.New("文档不存在")
	}
	if err := s.checkQuotaLocked(id, content); err != nil {
		return Doc{}, err
	}
	d := s.docs[i]
	d.Name = cleanName(name, d.Kind)
	d.Content = content
	d.Size = utf8.RuneCountInString(content)
	d.UpdatedAt = time.Now().Unix()
	s.docs[i] = d
	if err := s.saveLocked(); err != nil {
		return Doc{}, err
	}
	d.Content = ""
	return d, nil
}

// SetScope 修改文档的使用范围（面试 / 搜题 / 两者）
func (s *Store) SetScope(id, scope string) error {
	if !validScope(scope) {
		return fmt.Errorf("无效的使用范围: %q", scope)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.indexLocked(id)
	if i < 0 {
		return errors.New("文档不存在")
	}
	if s.docs[i].Scope == scope {
		return nil
	}
	s.docs[i].Scope = scope
	return s.saveLocked()
}

// Delete 删除文档
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.indexLocked(id)
	if i < 0 {
		return errors.New("文档不存在")
	}
	s.docs = append(s.docs[:i], s.docs[i+1:]...)
	return s.saveLocked()
}

// Remaining 返回当前剩余可用字数
func (s *Store) Remaining() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return MaxTotalRunes - s.usedLocked(-1)
}

// ==================== 字数上限 ====================

// OverLimitError 总字数超过上限
type OverLimitError struct {
	Used  int // 其它文档已占用的字数
	Need  int // 本次要写入的字数
	Limit int
}

func (e *OverLimitError) Error() string {
	return fmt.Sprintf("超出知识库字数上限：其它文档已用 %d 字，本次内容 %d 字，合计将超过 %d 字上限（超出 %d 字）。请先删减其它文档或精简内容",
		e.Used, e.Need, e.Limit, e.Used+e.Need-e.Limit)
}

// checkQuotaLocked 校验写入 content（替换 ignoreID 文档）后是否仍在上限内
func (s *Store) checkQuotaLocked(ignoreID, content string) error {
	used := s.usedLocked(s.indexLocked(ignoreID))
	need := utf8.RuneCountInString(content)
	if used+need > MaxTotalRunes {
		return &OverLimitError{Used: used, Need: need, Limit: MaxTotalRunes}
	}
	return nil
}

// usedLocked 统计总字数，skip 为要排除的下标（-1 不排除）
func (s *Store) usedLocked(skip int) int {
	n := 0
	for i, d := range s.docs {
		if i != skip {
			n += d.Size
		}
	}
	return n
}

// ==================== 内部 ====================

func (s *Store) indexLocked(id string) int {
	if id == "" {
		return -1
	}
	for i, d := range s.docs {
		if d.ID == id {
			return i
		}
	}
	return -1
}

// sortedLocked 简历在前，其余保持添加顺序（返回副本）
func (s *Store) sortedLocked() []Doc {
	out := make([]Doc, 0, len(s.docs))
	for _, d := range s.docs {
		if d.Kind == KindResume {
			out = append(out, d)
		}
	}
	for _, d := range s.docs {
		if d.Kind != KindResume {
			out = append(out, d)
		}
	}
	return out
}

type fileData struct {
	Docs []Doc `json:"docs"`
}

func (s *Store) load() {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Printf("[knowledge] 读取失败: %v", err)
		}
		return
	}
	plain, err := config.Decrypt(raw)
	if err != nil {
		logger.Printf("[knowledge] 解密失败: %v", err)
		return
	}
	var fd fileData
	if err := json.Unmarshal(plain, &fd); err != nil {
		logger.Printf("[knowledge] 解析失败: %v", err)
		return
	}
	for i := range fd.Docs {
		fd.Docs[i].Size = utf8.RuneCountInString(fd.Docs[i].Content)
		// 引入使用范围之前保存的文档没有 scope，按原有行为视为仅面试
		if !validScope(fd.Docs[i].Scope) {
			fd.Docs[i].Scope = ScopeInterview
		}
	}
	s.docs = fd.Docs
}

// saveLocked 先写临时文件再改名，避免写到一半崩溃损坏已有数据
func (s *Store) saveLocked() error {
	plain, err := json.Marshal(fileData{Docs: s.docs})
	if err != nil {
		return err
	}
	data, err := config.Encrypt(plain)
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func validScope(scope string) bool {
	return scope == ScopeInterview || scope == ScopeSolve || scope == ScopeBoth
}

func newID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// cleanName 去除首尾空白与换行，限制长度；为空时按类型给默认名
func cleanName(name, kind string) string {
	name = strings.Join(strings.Fields(name), " ")
	if r := []rune(name); len(r) > maxNameRunes {
		name = string(r[:maxNameRunes])
	}
	if name != "" {
		return name
	}
	switch kind {
	case KindResume:
		return "简历"
	case KindNote:
		return "未命名笔记"
	}
	return "未命名文档"
}

// Normalize 统一换行、去除 BOM 与首尾空白，并把连续 3 个以上空行压成 1 个空行
func Normalize(s string) string {
	s = strings.TrimPrefix(s, "\uFEFF")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.TrimSpace(s)
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return s
}
