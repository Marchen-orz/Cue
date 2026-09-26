package audio

import (
	"math"
	"testing"
)

// TestConverterRate 采样率转换输出数量应与理论值一致（±2）。
func TestConverterRate(t *testing.T) {
	cases := []struct {
		name    string
		srcRate int
		input   int // 输入单声道样本数
	}{
		{"48k->16k", 48000, 48000},
		{"44.1k->16k", 44100, 44100},
		{"96k->16k", 96000, 48000},
		{"16k->16k", 16000, 16000},
	}
	for _, tc := range cases {
		c := NewConverter(tc.srcRate, 1, false)
		// 构造正弦波输入分块喂入
		input := make([]float64, tc.input)
		for i := range input {
			input[i] = math.Sin(float64(i) * 0.01)
		}
		var total int
		chunk := tc.srcRate / 10 // 100ms 分块
		for start := 0; start < len(input); start += chunk {
			end := start + chunk
			if end > len(input) {
				end = len(input)
			}
			total += len(c.resample(input[start:end]))
		}
		want := tc.input * SampleRate / tc.srcRate
		if abs(total-want) > 2 {
			t.Errorf("%s: 输出 %d，期望 %d（±2）", tc.name, total, want)
		} else {
			t.Logf("%s: 输出 %d（期望 %d）✓", tc.name, total, want)
		}
	}
}

// TestConverterSineFrequency 正弦波降采样后频率应保持（过零点间隔校验）。
func TestConverterSineFrequency(t *testing.T) {
	const srcRate = 48000
	const freq = 440.0 // 440Hz
	c := NewConverter(srcRate, 1, false)

	n := srcRate // 1 秒
	input := make([]float64, n)
	for i := range input {
		input[i] = math.Sin(2 * math.Pi * freq * float64(i) / srcRate)
	}
	out := c.Convert(int16Interleave(input))
	if len(out) < SampleRate-20 {
		t.Fatalf("输出样本过少: %d", len(out))
	}
	// 统计正半周数量（440Hz 在 1 秒内有 440 个正半周）
	cycles := 0
	prevPositive := false
	for _, v := range out {
		positive := v > 0
		if positive && !prevPositive {
			cycles++
		}
		prevPositive = positive
	}
	if abs(cycles-440) > 8 {
		t.Errorf("过零周期 %d，期望约 440", cycles)
	} else {
		t.Logf("440Hz 正弦波检出 %d 个周期 ✓", cycles)
	}
}

// TestConverterSilencePadding 补零逻辑：静音块数量应与输入时长匹配
func TestConverterSilencePadding(t *testing.T) {
	c := NewConverter(48000, 2, false)
	// 100ms 的 48k 双声道 int16 静音
	frames := 4800
	buf := make([]byte, frames*2*2)
	out := c.Convert(buf)
	want := frames * SampleRate / 48000
	if abs(len(out)-want) > 2 {
		t.Errorf("输出 %d，期望 %d", len(out), want)
	}
	for _, v := range out {
		if v != 0 {
			t.Fatalf("静音输入产生非零输出: %d", v)
		}
	}
}

// TestConverterFloatScale float32 输入（WASAPI 共享模式格式，范围 ±1）
// 必须缩放到 int16 量程，否则所有样本截断为 0。
func TestConverterFloatScale(t *testing.T) {
	c := NewConverter(48000, 2, true)
	frames := 4800
	buf := make([]byte, frames*2*4)
	for i := 0; i < frames; i++ {
		v := float32(0.5 * math.Sin(2*math.Pi*440*float64(i)/48000))
		bits := math.Float32bits(v)
		for ch := 0; ch < 2; ch++ {
			off := (i*2 + ch) * 4
			buf[off] = byte(bits)
			buf[off+1] = byte(bits >> 8)
			buf[off+2] = byte(bits >> 16)
			buf[off+3] = byte(bits >> 24)
		}
	}
	out := c.Convert(buf)
	// 振幅 0.5 的正弦 RMS ≈ 0.5/√2 * 32767 ≈ 11585
	if r := RMS(out); r < 10000 || r > 13000 {
		t.Errorf("float 输入 RMS=%.1f，期望约 11585", r)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// int16Interleave 把 float64 单声道转成 int16 单声道字节流（模拟采集格式）
func int16Interleave(mono []float64) []byte {
	buf := make([]byte, len(mono)*2)
	for i, v := range mono {
		s := clampInt16(v * 30000)
		buf[i*2] = byte(uint16(s) & 0xff)
		buf[i*2+1] = byte(uint16(s) >> 8)
	}
	return buf
}
