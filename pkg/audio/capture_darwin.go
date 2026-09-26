//go:build darwin

package audio

import (
	"context"
	"fmt"
	"sync"
)

// darwinCapturer v1 仅支持麦克风采集（AVAudioEngine）。
// 系统音频回环需要 ScreenCaptureKit（macOS 13+）+ 屏幕录制权限，暂未实现。
type darwinCapturer struct {
	source Source

	mu      sync.Mutex
	running bool
}

func newPlatformCapturer(src Source) (Capturer, error) {
	if src == SourceSystem {
		return nil, fmt.Errorf("macOS 暂不支持系统音频采集")
	}
	return &darwinCapturer{source: src}, nil
}

func (d *darwinCapturer) Start(ctx context.Context) (<-chan []int16, <-chan error, error) {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return nil, nil, fmt.Errorf("采集器已在运行")
	}
	d.running = true
	d.mu.Unlock()

	stream := make(chan []int16, 32)
	errCh := make(chan error, 4)

	go func() {
		defer func() {
			d.mu.Lock()
			d.running = false
			d.mu.Unlock()
			close(stream)
			close(errCh)
		}()
		if err := d.run(ctx, stream, errCh); err != nil && ctx.Err() == nil {
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	return stream, errCh, nil
}

// run TODO: 接入 AVAudioEngine cgo 实现（16k 单声道 tap）。
// 当前返回错误占位，阶段 4 实现。
func (d *darwinCapturer) run(ctx context.Context, stream chan<- []int16, errCh chan<- error) error {
	return fmt.Errorf("macOS 麦克风采集将在后续版本支持")
}
