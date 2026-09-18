//go:build prod

package logger

import (
	"log"
	"os"
	"path/filepath"

	"Q-Solver/pkg/common"
)

// prod 构建无控制台，日志落盘到 %APPDATA%/Q-Solver/Q-Solver.log
// 每次启动清空重写，便于定位本次会话的问题
func init() {
	dir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	appDir := filepath.Join(dir, common.AppName)
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(appDir, "Q-Solver.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}

func Println(v ...interface{}) {
	log.Println(v...)
}

func Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func Info(v ...interface{}) {
	log.Println(append([]interface{}{"[INFO]"}, v...)...)
}

func Error(v ...interface{}) {
	log.Println(append([]interface{}{"[ERROR]"}, v...)...)
}
