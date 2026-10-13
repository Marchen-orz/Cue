package common

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, p, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), 0644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestMigrateDirRename(t *testing.T) {
	root := t.TempDir()
	oldD, newD := filepath.Join(root, "old"), filepath.Join(root, "new")
	write(t, filepath.Join(oldD, "config"), "cfg")
	write(t, filepath.Join(oldD, "skills", "a", "SKILL.md"), "skill")

	MigrateDir(oldD, newD)

	if read(t, filepath.Join(newD, "config")) != "cfg" || read(t, filepath.Join(newD, "skills", "a", "SKILL.md")) != "skill" {
		t.Fatal("data not migrated")
	}
	if _, err := os.Stat(oldD); !os.IsNotExist(err) {
		t.Fatal("old dir should be gone")
	}
}

func TestMigrateDirMerge(t *testing.T) {
	root := t.TempDir()
	oldD, newD := filepath.Join(root, "old"), filepath.Join(root, "new")
	write(t, filepath.Join(oldD, "config"), "old-cfg")
	write(t, filepath.Join(oldD, "skills", "a", "SKILL.md"), "skill")
	write(t, filepath.Join(newD, "Cue.log"), "log") // 日志模块已提前建好新目录
	write(t, filepath.Join(newD, "config"), "new-cfg")

	MigrateDir(oldD, newD)

	if read(t, filepath.Join(newD, "config")) != "new-cfg" {
		t.Fatal("existing new data must be kept")
	}
	if read(t, filepath.Join(newD, "skills", "a", "SKILL.md")) != "skill" {
		t.Fatal("non-conflicting data must be moved")
	}
	// 冲突项留在旧目录，不丢数据
	if read(t, filepath.Join(oldD, "config")) != "old-cfg" {
		t.Fatal("conflicting old data must be preserved")
	}
}

func TestMigrateDirNoOld(t *testing.T) {
	root := t.TempDir()
	MigrateDir(filepath.Join(root, "missing"), filepath.Join(root, "new"))
	if _, err := os.Stat(filepath.Join(root, "new")); !os.IsNotExist(err) {
		t.Fatal("must not create new dir when nothing to migrate")
	}
}
