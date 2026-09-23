//go:build !windows

package app

import (
	"os"
	"path/filepath"
)

// documentsDir 获取用户文档目录（非 Windows 回退到 home/Documents）
func documentsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, "Documents")
}
