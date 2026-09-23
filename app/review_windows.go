//go:build windows

package app

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// documentsDir 获取系统的"文档"目录（支持 OneDrive 重定向）
func documentsDir() string {
	dir, err := windows.KnownFolderPath(windows.FOLDERID_Documents, windows.KF_FLAG_DEFAULT)
	if err != nil || dir == "" {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return "."
		}
		return filepath.Join(home, "Documents")
	}
	return dir
}
