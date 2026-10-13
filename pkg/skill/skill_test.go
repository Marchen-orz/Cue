package skill

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSkillMD(t *testing.T) {
	name, desc, body := ParseSkillMD("---\nname: Java 并发\ndescription: >\n  回答 Java 并发\n  与锁相关的问题\n---\n# 正文\n要点", "x")
	if name != "Java 并发" || desc != "回答 Java 并发 与锁相关的问题" || body != "# 正文\n要点" {
		t.Fatalf("got %q %q %q", name, desc, body)
	}
	name, desc, _ = ParseSkillMD("# 标题\n内容", "fallback")
	if name != "fallback" || desc != "标题" {
		t.Fatalf("fallback: %q %q", name, desc)
	}
}

func TestStoreLifecycle(t *testing.T) {
	st := &Store{dir: t.TempDir()}
	src := filepath.Join(t.TempDir(), "SKILL.md")
	os.WriteFile(src, []byte("---\nname: Go Concurrency\ndescription: goroutine questions\n---\nbody"), 0644)

	sk, err := st.ImportFile(src)
	if err != nil || sk.ID != "go-concurrency" || !sk.Enabled {
		t.Fatalf("import: %+v %v", sk, err)
	}
	if err := st.SetEnabled(sk.ID, false); err != nil {
		t.Fatal(err)
	}
	if len(st.Enabled()) != 0 || len(st.List()) != 1 {
		t.Fatal("disable not applied")
	}
	// 重新导入同名技能保留停用状态
	if sk, _ = st.ImportFile(src); sk.Enabled {
		t.Fatal("re-import should keep disabled state")
	}
	if _, err := st.Get("../x"); err == nil {
		t.Fatal("path traversal must be rejected")
	}
	if err := st.Delete(sk.ID); err != nil || len(st.List()) != 0 {
		t.Fatalf("delete: %v", err)
	}
}

func TestImportZip(t *testing.T) {
	zp := filepath.Join(t.TempDir(), "s.zip")
	f, _ := os.Create(zp)
	zw := zip.NewWriter(f)
	w, _ := zw.Create("pkg/deep/SKILL.md")
	w.Write([]byte("---\nname: deep\n---\ndeep body"))
	w, _ = zw.Create("pkg/SKILL.md")
	w.Write([]byte("---\nname: shallow\n---\nshallow body"))
	zw.Close()
	f.Close()

	st := &Store{dir: t.TempDir()}
	sk, err := st.ImportFile(zp)
	if err != nil || sk.ID != "shallow" {
		t.Fatalf("zip: %+v %v", sk, err)
	}
}

func TestPickAndBlock(t *testing.T) {
	c := []Skill{{ID: "a", Name: "A", Body: "ba"}, {ID: "b", Name: "B", Body: "bb"}, {ID: "c", Name: "C", Body: "bc"}}
	got := pick("好的：[\"b\", \"zzz\", \"a\", \"c\"]", c)
	if len(got) != 2 || got[0].ID != "b" || got[1].ID != "a" {
		t.Fatalf("pick: %+v", got)
	}
	if pick("[]", c) != nil || pick("没有", c) != nil {
		t.Fatal("empty expected")
	}
	if !strings.Contains(Block(got), "bb") || Block(nil) != "" {
		t.Fatal("block")
	}
}
