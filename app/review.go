package app

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"Q-Solver/pkg/logger"
)

// reviewManager 存储复盘管理器：
// 开启后每次启动（或开启）创建一个会话文件夹，之后的每道题
// （截图 + 解答）写入该文件夹：题目N.md + images/题目N_*.png
type reviewManager struct {
	enabled atomic.Bool

	mu         sync.Mutex
	sessionDir string // 当前会话文件夹（空 = 尚未创建）
	counter    int    // 已写入的题目数
	// 待写解答的题目信息（beginQuestion 分配，saveSolution 消费）
	pendingIndex  int
	pendingImages []string
}

func newReviewManager() *reviewManager {
	return &reviewManager{}
}

// SetEnabled 开启/关闭复盘模式；开启时确保会话文件夹就绪
func (rm *reviewManager) SetEnabled(on bool) {
	rm.enabled.Store(on)
	if on {
		if err := rm.ensureSession(); err != nil {
			logger.Printf("[review] 创建复盘会话文件夹失败: %v", err)
		} else {
			logger.Printf("[review] 复盘模式已开启，会话文件夹: %s", rm.sessionDir)
		}
	} else {
		logger.Println("[review] 复盘模式已关闭")
	}
}

func (rm *reviewManager) IsEnabled() bool {
	return rm.enabled.Load()
}

// baseDir 复盘根目录：<文档>/Q-Solver/复盘
func (rm *reviewManager) baseDir() string {
	return filepath.Join(documentsDir(), "Q-Solver", "复盘")
}

// ensureSession 创建本次启动的会话文件夹（每次启动/开启一个）
func (rm *reviewManager) ensureSession() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if rm.sessionDir != "" {
		return nil
	}
	dir := filepath.Join(rm.baseDir(), time.Now().Format("2006-01-02_150405"))
	if err := os.MkdirAll(filepath.Join(dir, "images"), 0o755); err != nil {
		return err
	}
	rm.sessionDir = dir
	return nil
}

// beginQuestion 分配题号并把题目截图落盘，返回题号与图片相对路径
func (rm *reviewManager) beginQuestion(screenshots []string) (int, []string, error) {
	if err := rm.ensureSession(); err != nil {
		return 0, nil, err
	}
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.counter++
	qName := fmt.Sprintf("题目%02d", rm.counter)
	images := make([]string, 0, len(screenshots))
	for i, b64 := range screenshots {
		data, ext, err := decodeImageBase64(b64)
		if err != nil {
			logger.Printf("[review] 解码截图失败: %v", err)
			continue
		}
		rel := filepath.Join("images", fmt.Sprintf("%s_%d.%s", qName, i+1, ext))
		if err := os.WriteFile(filepath.Join(rm.sessionDir, rel), data, 0o644); err != nil {
			logger.Printf("[review] 写入截图失败: %v", err)
			continue
		}
		images = append(images, rel)
	}

	rm.pendingIndex = rm.counter
	rm.pendingImages = images
	return rm.counter, images, nil
}

// saveSolution 把解答 markdown 写入题目文档（引用题目图片）
func (rm *reviewManager) saveSolution(md string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if rm.sessionDir == "" || rm.pendingIndex == 0 {
		return fmt.Errorf("没有待写入的复盘题目")
	}

	qName := fmt.Sprintf("题目%02d", rm.pendingIndex)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s\n\n", qName))
	b.WriteString(fmt.Sprintf("> 时间：%s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	if len(rm.pendingImages) > 0 {
		b.WriteString("## 题目截图\n\n")
		for i, img := range rm.pendingImages {
			b.WriteString(fmt.Sprintf("![题目截图 %d](%s)\n\n", i+1, filepath.ToSlash(img)))
		}
	}

	b.WriteString("## 解答\n\n")
	b.WriteString(md)
	b.WriteString("\n")

	if err := os.WriteFile(filepath.Join(rm.sessionDir, qName+".md"), []byte(b.String()), 0o644); err != nil {
		return err
	}

	rm.pendingIndex = 0
	rm.pendingImages = nil
	logger.Printf("[review] 已保存 %s.md", qName)
	return nil
}

// decodeImageBase64 解析 data URL（data:image/xxx;base64,....）
func decodeImageBase64(b64 string) ([]byte, string, error) {
	const marker = ";base64,"
	idx := strings.Index(b64, marker)
	if idx == -1 || !strings.HasPrefix(b64, "data:") {
		return nil, "", fmt.Errorf("不是有效的图片 data URL")
	}
	mime := strings.TrimPrefix(b64[:idx], "data:")
	ext := "png"
	if strings.Contains(mime, "jpeg") || strings.Contains(mime, "jpg") {
		ext = "jpg"
	} else if strings.Contains(mime, "webp") {
		ext = "webp"
	}
	data, err := base64.StdEncoding.DecodeString(b64[idx+len(marker):])
	if err != nil {
		return nil, "", err
	}
	return data, ext, nil
}
