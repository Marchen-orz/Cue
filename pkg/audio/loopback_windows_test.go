//go:build windows

package audio

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// writeWAVFile 生成 880Hz 正弦波 WAV 文件到指定路径
func writeWAVFile(path string) {
	const rate = 44100
	const dur = 2.5
	samples := int(rate * dur)
	pcm := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		v := int16(math.Sin(2*math.Pi*880*float64(i)/rate) * 20000)
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(v))
	}
	hdr := []byte("RIFF")
	size := 36 + len(pcm)
	hdr = binary.LittleEndian.AppendUint32(hdr, uint32(size))
	hdr = append(hdr, "WAVEfmt "...)
	hdr = binary.LittleEndian.AppendUint32(hdr, 16)
	hdr = binary.LittleEndian.AppendUint16(hdr, 1) // PCM
	hdr = binary.LittleEndian.AppendUint16(hdr, 1) // mono
	hdr = binary.LittleEndian.AppendUint32(hdr, rate)
	hdr = binary.LittleEndian.AppendUint32(hdr, rate*2)
	hdr = binary.LittleEndian.AppendUint16(hdr, 2)
	hdr = binary.LittleEndian.AppendUint16(hdr, 16)
	hdr = append(hdr, "data"...)
	hdr = binary.LittleEndian.AppendUint32(hdr, uint32(len(pcm)))
	_ = os.WriteFile(path, append(hdr, pcm...), 0o644)
}

// TestSystemLoopbackCaptureReal 真实采集 4 秒系统音频，
// 采集期间播放 2.5 秒 880Hz 正弦音，检验回环能收到播放内容。
func TestSystemLoopbackCaptureReal(t *testing.T) {
	c, err := NewCapturer(SourceSystem)
	if err != nil {
		t.Fatalf("创建采集器失败: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	stream, errCh, err := c.Start(ctx)
	if err != nil {
		t.Fatalf("启动采集失败: %v", err)
	}

	// 播放 WAV（走默认输出设备，应被回环捕获）。
	// 用固定路径的 WAV（PowerShell SoundPlayer 播放）。
	wavPath := os.Getenv("TEMP")
	if wavPath == "" {
		wavPath = t.TempDir()
	}
	wav := filepath.Join(wavPath, "qsolver_tone.wav")
	writeWAVFile(wav)
	player := exec.Command("powershell", "-NoProfile", "-Command",
		fmt.Sprintf("(New-Object System.Media.SoundPlayer '%s').PlaySync()", wav))
	if err := player.Start(); err != nil {
		t.Logf("播放失败（RMS 可能全为 0）: %v", err)
	}
	defer player.Wait()

	blocks := 0
	maxRMS := 0.0
	nonZero := 0
loop:
	for {
		select {
		case chunk, ok := <-stream:
			if !ok {
				break loop
			}
			blocks++
			if len(chunk) != ChunkSamples {
				t.Errorf("块长度错误: %d != %d", len(chunk), ChunkSamples)
			}
			r := RMS(chunk)
			if r > 100 {
				nonZero++
			}
			if r > maxRMS {
				maxRMS = r
			}
		case err := <-errCh:
			if err != nil {
				t.Fatalf("采集错误: %v", err)
			}
			break loop
		case <-ctx.Done():
			break loop
		}
	}
	if blocks < 35 {
		t.Errorf("4 秒内块数过少: %d（期望约 40）", blocks)
	}
	if nonZero == 0 {
		t.Errorf("未捕获到任何声音（最大 RMS=%.1f）——回环采集可能未生效", maxRMS)
	}
	t.Logf("收到 %d 块，最大 RMS=%.1f，有声块=%d", blocks, maxRMS, nonZero)
}
