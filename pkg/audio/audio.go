// Package audio 提供跨平台的音频采集，统一输出
// 16kHz / 16bit / 单声道 PCM，供 ASR 流式识别使用。
//
// 平台实现：
//   - windows: WASAPI 共享模式（麦克风采集 + 渲染端回环采集）
//   - darwin:  AVAudioEngine（仅麦克风；系统音频回环暂不支持）
package audio

import (
	"context"
	"fmt"
	"math"
	"runtime"
)

// Source 音频来源
type Source string

const (
	// SourceSystem 系统播放声音（渲染端回环采集）
	SourceSystem Source = "system"
	// SourceMicrophone 麦克风
	SourceMicrophone Source = "mic"
)

// SampleRate 输出 PCM 的采样率
const SampleRate = 16000

// ChunkSamples 每个输出块的采样数（100ms @16kHz）
const ChunkSamples = 1600

// Capturer 音频采集器。
// Start 后通过返回的 channel 持续收到固定长度的 PCM 块
// （[]int16，单声道，长度恒为 ChunkSamples，静音时为全零块，
// 保证下游 100ms 一块的节奏恒定，可用于静音计时与 ASR 保活）。
// 采集异常通过 errCh 报告并终止流。
type Capturer interface {
	// Start 启动采集。ctx 取消时采集停止并关闭两个 channel。
	Start(ctx context.Context) (stream <-chan []int16, errCh <-chan error, err error)
}

// NewCapturer 按来源创建采集器。仅限主线程外使用的纯 Go 接口。
func NewCapturer(src Source) (Capturer, error) {
	switch src {
	case SourceSystem:
		if runtime.GOOS != "windows" {
			return nil, fmt.Errorf("%s 平台暂不支持系统音频采集（回环）", runtime.GOOS)
		}
	case SourceMicrophone:
	default:
		return nil, fmt.Errorf("未知音频来源: %s", src)
	}
	return newPlatformCapturer(src)
}

// ==================== PCM 转换 ====================

// Converter 把采集到的原始交错 PCM（float32 或 int16、任意声道、任意采样率）
// 流式转换为 16kHz 单声道 int16。内部持有重采样进度，跨块连续。
type Converter struct {
	srcRate  int
	channels int
	isFloat  bool

	ratio   float64 // srcRate / SampleRate
	prev    float64 // 上一个已消费输入样本值
	hasPrev bool
	absIdx  float64 // 已消费输入样本的绝对索引
	nextOut float64 // 下一个输出样本的绝对位置（单位：输入样本间隔）
}

// NewConverter 创建转换器。srcRate/channels 来自 WASAPI MixFormat。
func NewConverter(srcRate, channels int, isFloat bool) *Converter {
	if srcRate <= 0 {
		srcRate = SampleRate
	}
	if channels <= 0 {
		channels = 1
	}
	return &Converter{
		srcRate:  srcRate,
		channels: channels,
		isFloat:  isFloat,
		ratio:    float64(srcRate) / float64(SampleRate),
	}
}

// Convert 转换一块交错 PCM 数据，返回单声道 16kHz int16 样本。
func (c *Converter) Convert(chunk []byte) []int16 {
	mono := c.mixdown(chunk)
	return c.resample(mono)
}

// mixdown 交错多声道 → 单声道 float64
func (c *Converter) mixdown(chunk []byte) []float64 {
	var out []float64
	if c.isFloat {
		n := len(chunk) / 4 / c.channels
		out = make([]float64, n)
		for i := 0; i < n; i++ {
			var sum float64
			for ch := 0; ch < c.channels; ch++ {
				bits := uint32(chunk[(i*c.channels+ch)*4]) |
					uint32(chunk[(i*c.channels+ch)*4+1])<<8 |
					uint32(chunk[(i*c.channels+ch)*4+2])<<16 |
					uint32(chunk[(i*c.channels+ch)*4+3])<<24
				sum += float64(math.Float32frombits(bits))
			}
			// float 采样范围 [-1, 1]，缩放到 int16 量程
			out[i] = sum / float64(c.channels) * 32767
		}
	} else {
		n := len(chunk) / 2 / c.channels
		out = make([]float64, n)
		for i := 0; i < n; i++ {
			var sum int32
			for ch := 0; ch < c.channels; ch++ {
				sum += int32(int16(chunk[(i*c.channels+ch)*2]) | int16(chunk[(i*c.channels+ch)*2+1])<<8)
			}
			out[i] = float64(sum) / float64(c.channels)
		}
	}
	return out
}

// resample 任意采样率 → 16kHz，线性插值。
// 输出样本位于绝对输入位置 0, ratio, 2*ratio, ...，
// 落在 (i-1, i] 区间的输出在 prev(位置 i-1) 与 cur(位置 i) 之间插值。
func (c *Converter) resample(mono []float64) []int16 {
	out := make([]int16, 0, int(float64(len(mono))/c.ratio)+2)
	for _, cur := range mono {
		if !c.hasPrev {
			c.hasPrev = true
			c.prev = cur
			c.absIdx = 0
			c.nextOut = c.ratio
			out = append(out, clampInt16(cur)) // 位置 0
			continue
		}
		c.absIdx++
		for c.nextOut <= c.absIdx {
			frac := c.nextOut - (c.absIdx - 1)
			out = append(out, clampInt16(c.prev+(cur-c.prev)*frac))
			c.nextOut += c.ratio
		}
		c.prev = cur
	}
	return out
}

func clampInt16(v float64) int16 {
	if v > 32767 {
		return 32767
	}
	if v < -32768 {
		return -32768
	}
	return int16(v)
}

// RMS 计算一块 PCM 的均方根（0~32767），用于静音检测
func RMS(chunk []int16) float64 {
	if len(chunk) == 0 {
		return 0
	}
	var sum float64
	for _, v := range chunk {
		sum += float64(v) * float64(v)
	}
	return math.Sqrt(sum / float64(len(chunk)))
}
