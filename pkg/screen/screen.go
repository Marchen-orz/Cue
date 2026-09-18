package screen

import (
	imageutil "Q-Solver/pkg/ImageUtil"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"

	"github.com/kbinani/screenshot"
)

type PreviewResult struct {
	ImgBytes []byte `json:"imgBytes"`
	Base64   string `json:"base64"`
	Size     string `json:"size"`
}

// DisplayInfo 显示器信息（供设置下拉选择）
type DisplayInfo struct {
	Index   int `json:"index"`
	X       int `json:"x"`
	Y       int `json:"y"`
	W       int `json:"w"`
	H       int `json:"h"`
	Primary bool `json:"primary"`
}

// CaptureOptions 截图参数
type CaptureOptions struct {
	Quality       int
	Sharpen       float64
	Grayscale     bool
	NoCompression bool
	Mode          string // fullscreen / region
	ScreenIndex   int    // fullscreen 模式的显示器序号
	Region        Region // region 模式的框选区域
}

// Service 截图服务
type Service struct {
	ctx context.Context
}

func NewService() *Service {
	return &Service{}
}

// Startup 初始化服务
func (s *Service) Startup(ctx context.Context) {
	s.ctx = ctx
}

// ListDisplays 枚举当前所有显示器
func ListDisplays() []DisplayInfo {
	n := screenshot.NumActiveDisplays()
	out := make([]DisplayInfo, 0, n)
	for i := 0; i < n; i++ {
		b := screenshot.GetDisplayBounds(i)
		out = append(out, DisplayInfo{
			Index:   i,
			X:       b.Min.X,
			Y:       b.Min.Y,
			W:       b.Dx(),
			H:       b.Dy(),
			Primary: b.Min.X == 0 && b.Min.Y == 0,
		})
	}
	return out
}

// CapturePreview 获取当前截图的预览（Base64），预览即最终送模型的截图
func (s *Service) CapturePreview(opts CaptureOptions) (PreviewResult, error) {
	var img *image.RGBA
	var err error

	switch opts.Mode {
	case "region":
		r := opts.Region.Normalized()
		if r.Empty() {
			return PreviewResult{}, fmt.Errorf("尚未设置截图区域，请先在设置中框选")
		}
		img, err = screenshot.CaptureRect(image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H))
	default: // fullscreen
		idx := opts.ScreenIndex
		if idx < 0 || idx >= screenshot.NumActiveDisplays() {
			idx = 0
		}
		img, err = screenshot.CaptureDisplay(idx)
	}
	if err != nil {
		return PreviewResult{}, fmt.Errorf("截图失败: %v", err)
	}

	// 处理图片
	var imgBytes []byte
	var ImageBase64 string
	if opts.NoCompression {
		var buf bytes.Buffer
		err = png.Encode(&buf, img)
		if err != nil {
			return PreviewResult{}, fmt.Errorf("图片编码失败: %v", err)
		}
		imgBytes = buf.Bytes()
		ImageBase64 = fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(imgBytes))
	} else {
		imgBytes, err = imageutil.CompressForOCR(img, opts.Quality, opts.Sharpen, opts.Grayscale)
		if err != nil {
			return PreviewResult{}, fmt.Errorf("图片处理失败: %v", err)
		}
		ImageBase64 = fmt.Sprintf("data:image/jpeg;base64,%s", base64.StdEncoding.EncodeToString(imgBytes))
	}

	// 计算大小
	sizeKB := float64(len(ImageBase64)) / 1024.0
	sizeStr := fmt.Sprintf("%.2f KB", sizeKB)

	// 转 Base64
	return PreviewResult{
		ImgBytes: imgBytes,
		Base64:   ImageBase64,
		Size:     sizeStr,
	}, nil
}
