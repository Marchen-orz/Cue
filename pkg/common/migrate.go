package common

import (
	"os"
	"path/filepath"
)

// init 在任何依赖本包的包（日志、配置）初始化之前运行：
// 把旧产品名下的数据目录（config、skills、日志）迁移到新目录，
// 避免改名后用户的 API Key、简历、技能丢失。
func init() {
	base, err := os.UserConfigDir()
	if err != nil {
		return
	}
	MigrateDir(filepath.Join(base, LegacyAppName), filepath.Join(base, AppName))
}

// MigrateDir 把 oldDir 迁移到 newDir：
//   - oldDir 不存在：什么都不做
//   - newDir 不存在：整体重命名（原子）
//   - newDir 已存在（例如日志模块已提前创建）：逐项合并，新目录里已有的同名项保留，
//     旧目录里不冲突的项移过来；旧目录清空后删除
//
// 任何一步失败都静默放弃，保留旧数据原样，不影响启动。
func MigrateDir(oldDir, newDir string) {
	if oldDir == newDir {
		return
	}
	st, err := os.Stat(oldDir)
	if err != nil || !st.IsDir() {
		return
	}
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		if os.Rename(oldDir, newDir) == nil {
			return
		}
	}
	if err := os.MkdirAll(newDir, 0755); err != nil {
		return
	}
	entries, err := os.ReadDir(oldDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		dst := filepath.Join(newDir, e.Name())
		if _, err := os.Stat(dst); err == nil {
			continue // 新目录已有同名项，保留新的
		}
		_ = os.Rename(filepath.Join(oldDir, e.Name()), dst)
	}
	_ = os.Remove(oldDir) // 仅在已清空时成功
}
