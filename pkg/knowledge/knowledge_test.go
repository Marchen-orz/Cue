package knowledge

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return &Store{path: filepath.Join(t.TempDir(), "knowledge")}
}

func TestStoreAddListGetDelete(t *testing.T) {
	s := newTestStore(t)

	d, err := s.Add("Go 笔记", KindNote, "  goroutine 调度\r\n\r\n\r\n\r\nGMP 模型  ")
	if err != nil {
		t.Fatal(err)
	}
	if d.Content != "" {
		t.Error("Add 的返回值不应带正文")
	}
	got, err := s.Get(d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "goroutine 调度\n\nGMP 模型" {
		t.Errorf("正文未被规范化: %q", got.Content)
	}
	if sum := s.List(); sum.Used != got.Size || sum.Limit != MaxTotalRunes || len(sum.Docs) != 1 || sum.Docs[0].Content != "" {
		t.Errorf("List 结果异常: %+v", sum)
	}

	if err := s.Delete(d.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(d.ID); err == nil {
		t.Error("删除后不应还能读到")
	}
	if s.Prompt(ScopeInterview) != "" {
		t.Error("知识库为空时 Prompt 应为空串")
	}
}

func TestStorePersistsEncrypted(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Add("简历", KindResume, "张三，十年 Go 后端经验"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("张三")) {
		t.Error("落盘内容应当是加密的")
	}

	s2 := &Store{path: s.path}
	s2.load()
	if sum := s2.List(); len(sum.Docs) != 1 || sum.Docs[0].Name != "简历" || sum.Used == 0 {
		t.Errorf("重新加载后数据丢失: %+v", sum)
	}
}

func TestStoreQuota(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Add("a", KindDoc, strings.Repeat("字", MaxTotalRunes-10)); err != nil {
		t.Fatal(err)
	}

	// 超出上限被拒绝，且不改变已有内容
	_, err := s.Add("b", KindDoc, strings.Repeat("字", 11))
	var over *OverLimitError
	if !errors.As(err, &over) {
		t.Fatalf("期望 OverLimitError，得到 %v", err)
	}
	if over.Used != MaxTotalRunes-10 || over.Need != 11 {
		t.Errorf("OverLimitError 数值异常: %+v", over)
	}
	if got := s.List(); len(got.Docs) != 1 || got.Used != MaxTotalRunes-10 {
		t.Errorf("被拒绝的写入不应落盘: %+v", got)
	}

	// 刚好用满是允许的
	if _, err := s.Add("c", KindDoc, strings.Repeat("字", 10)); err != nil {
		t.Errorf("刚好用满应当允许: %v", err)
	}
	if s.Remaining() != 0 {
		t.Errorf("剩余应为 0，得到 %d", s.Remaining())
	}

	// 修改自己时按"替换"计算：缩短始终允许，增长受限
	a := s.List().Docs[0]
	if _, err := s.Update(a.ID, "a", strings.Repeat("字", 100)); err != nil {
		t.Errorf("缩短应当允许: %v", err)
	}
	if _, err := s.Update(a.ID, "a", strings.Repeat("字", MaxTotalRunes)); err == nil {
		t.Error("增长超限应当被拒绝")
	}
}

func TestStoreResumeReplaced(t *testing.T) {
	s := newTestStore(t)
	// 简历占满额度后，再导入新简历是"替换"，不应因额度被拒
	first, err := s.Add("旧简历", KindResume, strings.Repeat("旧", MaxTotalRunes))
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Add("新简历", KindResume, "新的简历内容")
	if err != nil {
		t.Fatalf("替换简历不应受额度限制: %v", err)
	}
	if second.ID != first.ID {
		t.Error("替换应沿用原文档 ID")
	}
	sum := s.List()
	if len(sum.Docs) != 1 || sum.Docs[0].Name != "新简历" {
		t.Errorf("知识库里应只有一份简历: %+v", sum)
	}
}

func TestPromptOrderAndFormat(t *testing.T) {
	s := newTestStore(t)
	mustAdd := func(name, kind, content string) {
		t.Helper()
		if _, err := s.Add(name, kind, content); err != nil {
			t.Fatal(err)
		}
	}
	mustAdd("项目笔记", KindNote, "订单系统拆分")
	mustAdd("简历", KindResume, "姓名：张三")

	p := s.Prompt(ScopeInterview)
	if !strings.HasPrefix(p, "\n\n<KnowledgeBase>") || !strings.HasSuffix(p, "</KnowledgeBase>\n") {
		t.Fatalf("Prompt 外层格式异常: %q", p)
	}
	if strings.Index(p, "姓名：张三") > strings.Index(p, "订单系统拆分") {
		t.Error("简历应排在其它文档之前")
	}
	if !strings.Contains(p, `kind="resume"`) || !strings.Contains(p, `name="项目笔记"`) {
		t.Errorf("缺少文档属性: %s", p)
	}
}

func TestScope(t *testing.T) {
	s := newTestStore(t)
	note, err := s.Add("笔记", KindNote, "订单系统拆分")
	if err != nil {
		t.Fatal(err)
	}
	if note.Scope != ScopeInterview {
		t.Errorf("新文档默认应仅用于面试，得到 %q", note.Scope)
	}
	if s.Prompt(ScopeSolve) != "" {
		t.Error("默认文档不应出现在搜题提示词里")
	}

	for _, c := range []struct {
		scope            string
		interview, solve bool
	}{
		{ScopeSolve, false, true},
		{ScopeBoth, true, true},
		{ScopeInterview, true, false},
	} {
		if err := s.SetScope(note.ID, c.scope); err != nil {
			t.Fatal(err)
		}
		if got := s.Prompt(ScopeInterview) != ""; got != c.interview {
			t.Errorf("scope=%s 面试提示词非空 = %v, want %v", c.scope, got, c.interview)
		}
		if got := s.Prompt(ScopeSolve) != ""; got != c.solve {
			t.Errorf("scope=%s 搜题提示词非空 = %v, want %v", c.scope, got, c.solve)
		}
	}

	if err := s.SetScope(note.ID, "nope"); err == nil {
		t.Error("无效范围应报错")
	}
	if err := s.SetScope("missing", ScopeBoth); err == nil {
		t.Error("文档不存在应报错")
	}

	// 修改正文与重新加载都不应丢失范围
	if err := s.SetScope(note.ID, ScopeSolve); err != nil {
		t.Fatal(err)
	}
	if d, err := s.Update(note.ID, "笔记", "新的正文"); err != nil || d.Scope != ScopeSolve {
		t.Errorf("Update 后范围丢失: %+v %v", d, err)
	}
	s2 := &Store{path: s.path}
	s2.load()
	if sum := s2.List(); len(sum.Docs) != 1 || sum.Docs[0].Scope != ScopeSolve {
		t.Errorf("重新加载后范围丢失: %+v", sum)
	}
}

func TestScopeLegacyAndResumeReplace(t *testing.T) {
	s := newTestStore(t)
	// 模拟引入范围之前保存的数据：没有 scope 字段
	s.docs = []Doc{{ID: "old", Name: "旧文档", Kind: KindDoc, Content: "内容"}}
	if err := s.saveLocked(); err != nil {
		t.Fatal(err)
	}
	s2 := &Store{path: s.path}
	s2.load()
	if got := s2.List().Docs[0].Scope; got != ScopeInterview {
		t.Errorf("旧数据应视为仅面试，得到 %q", got)
	}

	// 替换简历时沿用用户已选的范围
	r, err := s2.Add("简历", KindResume, "第一版")
	if err != nil {
		t.Fatal(err)
	}
	if err := s2.SetScope(r.ID, ScopeBoth); err != nil {
		t.Fatal(err)
	}
	r2, err := s2.Add("简历", KindResume, "第二版")
	if err != nil {
		t.Fatal(err)
	}
	if r2.ID != r.ID || r2.Scope != ScopeBoth {
		t.Errorf("替换简历应沿用 ID 与范围: %+v", r2)
	}
}

func TestCleanName(t *testing.T) {
	cases := []struct{ name, kind, want string }{
		{"  a \n b  ", KindDoc, "a b"},
		{"", KindResume, "简历"},
		{"   ", KindNote, "未命名笔记"},
		{"", KindDoc, "未命名文档"},
		{strings.Repeat("长", 100), KindDoc, strings.Repeat("长", maxNameRunes)},
	}
	for _, c := range cases {
		if got := cleanName(c.name, c.kind); got != c.want {
			t.Errorf("cleanName(%q, %s) = %q, want %q", c.name, c.kind, got, c.want)
		}
	}
}

func TestExtractLocalText(t *testing.T) {
	dir := t.TempDir()

	utf8Path := filepath.Join(dir, "a.md")
	if err := os.WriteFile(utf8Path, []byte("\xef\xbb\xbf# 标题\n正文"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := ExtractLocal(utf8Path)
	if err != nil || !strings.Contains(got, "# 标题") {
		t.Errorf("UTF-8 读取失败: %q %v", got, err)
	}

	gbk, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte("面试八股文：并发与锁"))
	if err != nil {
		t.Fatal(err)
	}
	gbkPath := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(gbkPath, gbk, 0644); err != nil {
		t.Fatal(err)
	}
	if got, err := ExtractLocal(gbkPath); err != nil || got != "面试八股文：并发与锁" {
		t.Errorf("GBK 回退失败: %q %v", got, err)
	}

	if _, err := ExtractLocal(filepath.Join(dir, "c.doc")); err == nil {
		t.Error("不支持的类型应报错")
	}
}

func TestExtractDocx(t *testing.T) {
	const docXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>
<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>个人技能</w:t></w:r></w:p>
<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>熟悉 </w:t></w:r><w:r><w:t>Go</w:t></w:r></w:p>
<w:p><w:r><w:t>第一行</w:t><w:br/><w:t>第二行</w:t></w:r></w:p>
<w:tbl><w:tr><w:tc><w:p><w:r><w:t>项目</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>成果</w:t></w:r></w:p></w:tc></w:tr>
<w:tr><w:tc><w:p><w:r><w:t>订单系统</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>QPS 翻倍</w:t></w:r></w:p></w:tc></w:tr></w:tbl>
</w:body></w:document>`

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("word/document.xml")
	_, _ = w.Write([]byte(docXML))
	_ = zw.Close()

	got, err := extractDocx(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# 个人技能", "- 熟悉 Go", "第一行\n第二行", "项目 | 成果", "订单系统 | QPS 翻倍"} {
		if !strings.Contains(got, want) {
			t.Errorf("DOCX 提取结果缺少 %q，实际:\n%s", want, got)
		}
	}

	if _, err := extractDocx([]byte("not a zip")); err == nil {
		t.Error("损坏的 DOCX 应报错")
	}
}
