//go:build windows

package audio

import (
	"context"
	"testing"
	"time"
)

// TestMicCaptureReal 真实采集 2 秒麦克风音频，打印每块 RMS。
// 无声环境下 RMS 应接近 0 但不为 NaN；对着麦克风说话 RMS 会明显升高。
func TestMicCaptureReal(t *testing.T) {
	c, err := NewCapturer(SourceMicrophone)
	if err != nil {
		t.Fatalf("创建采集器失败: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, errCh, err := c.Start(ctx)
	if err != nil {
		t.Fatalf("启动采集失败: %v", err)
	}

	blocks := 0
	maxRMS := 0.0
	for {
		select {
		case chunk, ok := <-stream:
			if !ok {
				goto done
			}
			blocks++
			if len(chunk) != ChunkSamples {
				t.Errorf("块长度错误: %d != %d", len(chunk), ChunkSamples)
			}
			r := RMS(chunk)
			if r > maxRMS {
				maxRMS = r
			}
		case err := <-errCh:
			if err != nil {
				t.Fatalf("采集错误: %v", err)
			}
			goto done
		case <-ctx.Done():
			goto done
		}
	}
done:
	if blocks < 15 {
		t.Errorf("2 秒内块数过少: %d（期望约 20）", blocks)
	}
	t.Logf("收到 %d 块，最大 RMS=%.1f", blocks, maxRMS)
}
