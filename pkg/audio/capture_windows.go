//go:build windows

package audio

import (
	"context"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

// audioDebug 采集调试开关（QSAUDIO_DEBUG=1 时输出诊断日志到 stderr）
var audioDebug = os.Getenv("QSAUDIO_DEBUG") == "1"

func debugf(format string, args ...interface{}) {
	if audioDebug {
		fmt.Fprintf(os.Stderr, "[audio] "+format+"\n", args...)
	}
}

// ==================== WASAPI COM 常量 ====================

const (
	clsidMMDeviceEnumerator = "{BCDE0395-E52F-467C-8E3D-C4579291692E}"
	iidIMMDeviceEnumerator  = "{A95664D2-9614-4F35-A746-DE8DB63617E6}"
	iidIAudioClient         = "{1CB9AD4C-DBFA-4c32-B178-C2F568A703B2}"
	iidIAudioCaptureClient  = "{C8ADBD64-E71E-48a0-A4DE-185C395CD317}"
	iidIAudioEndpointVolume = "{5CDF2C82-841E-4546-9722-0CF74078229A}"
	subFormatIEEEFloatGUID  = "{00000003-0000-0010-8000-00aa00389b71}"

	eRender  = 0 // 渲染设备（用于回环采集）
	eCapture = 1 // 捕获设备（麦克风）
	eConsole = 0 // 控制台角色（默认设备）

	AUDCLNT_SHAREMODE_SHARED = 0
	// 回环采集标志：在渲染设备上共享模式采集。
	// 注意回环采集不支持事件驱动，必须轮询。
	AUDCLNT_STREAMFLAGS_LOOPBACK = 0x00020000

	AUDCLNT_BUFFERFLAGS_SILENT = 0x2

	REFTIMES_PER_SEC = 10000000 // 1 秒，单位 100ns

	// WAVE_FORMAT 常量
	waveFormatIEEEFloat = 3
	waveFormatExtensible = 0xFFFE

	// vtable 槽位（IUnknown 占 0/1/2）
	vtEnumerator_GetDefaultAudioEndpoint = 4
	vtEnumerator_EnumAudioEndpoints      = 3

	vtDevice_Activate         = 3
	vtDevice_OpenPropertyStore = 4
	vtDevice_GetId            = 5

	vtClient_Initialize                  = 3
	vtClient_GetMixFormat                = 8
	vtClient_Start                       = 10
	vtClient_Stop                        = 11
	vtClient_GetService                  = 14
	vtCapture_GetBuffer                  = 3
	vtCapture_ReleaseBuffer              = 4
	vtCapture_GetNextPacketSize          = 5
	vtUnknown_Release                    = 2

	// IPropertyStore
	vtPropStore_GetValue = 5

	// IMMDeviceCollection
	vtCollection_GetCount = 3
	vtCollection_Item     = 4

	// IAudioEndpointVolume
	vtEndpointVol_GetMasterVolumeLevelScalar = 9
	vtEndpointVol_GetMute                    = 15
)

const coinitFlags = 0x0 // COINIT_MULTITHREADED

var (
	ole32 = syscall.NewLazyDLL("ole32.dll")
	procCoInitializeEx   = ole32.NewProc("CoInitializeEx")
	procCoCreateInstance = ole32.NewProc("CoCreateInstance")
	procCoTaskMemFree    = ole32.NewProc("CoTaskMemFree")
	procIIDFromString    = ole32.NewProc("IIDFromString")
)

// ==================== 结构体 ====================

type windowsGUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type waveFormatEx struct {
	wFormatTag      uint16
	nChannels       uint16
	nSamplesPerSec  uint32
	nAvgBytesPerSec uint32
	nBlockAlign     uint16
	wBitsPerSample  uint16
	cbSize          uint16
}

func guidFromString(s string) (g windowsGUID, err error) {
	p, _ := syscall.UTF16PtrFromString(s)
	// IIDFromString(lpsz, lpiid)：第一个参数是字符串，第二个是输出 GUID；
	// 成功返回 S_OK(0)，失败返回负 HRESULT
	r, _, _ := procIIDFromString.Call(
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&g)),
	)
	if r == 0 {
		return g, nil
	}
	return g, fmt.Errorf("无效 GUID: %s (hr=0x%08x)", s, uint32(r))
}

// callCOM 通过对象 vtable 槽位调用 COM 方法
func callCOM(obj unsafe.Pointer, slot int, args ...uintptr) uintptr {
	vtbl := *(*unsafe.Pointer)(obj)
	fn := *(*unsafe.Pointer)(unsafe.Pointer(uintptr(vtbl) + uintptr(slot)*unsafe.Sizeof(uintptr(0))))
	full := append([]uintptr{uintptr(obj)}, args...)
	r, _, _ := syscall.SyscallN(uintptr(fn), full...)
	return r
}

func releaseCOM(obj unsafe.Pointer) {
	if obj != nil {
		callCOM(obj, vtUnknown_Release)
	}
}

// ==================== wasapiCapturer ====================

// wasapiCapturer WASAPI 共享模式轮询采集器（麦克风 / 渲染端回环）
type wasapiCapturer struct {
	source Source

	mu      sync.Mutex
	running bool
}

func newPlatformCapturer(src Source) (Capturer, error) {
	return &wasapiCapturer{source: src}, nil
}

func (w *wasapiCapturer) Start(ctx context.Context) (<-chan []int16, <-chan error, error) {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil, nil, fmt.Errorf("采集器已在运行")
	}
	w.running = true
	w.mu.Unlock()

	stream := make(chan []int16, 32)
	errCh := make(chan error, 4)

	go func() {
		defer func() {
			w.mu.Lock()
			w.running = false
			w.mu.Unlock()
			close(stream)
			close(errCh)
		}()
		if err := w.run(ctx, stream, errCh); err != nil && ctx.Err() == nil {
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	return stream, errCh, nil
}

// ensureCOM 初始化 COM（MTA 模式，重复调用安全）
func ensureCOM() {
	procCoInitializeEx.Call(0, coinitFlags)
}

// run 采集主循环。失败返回错误；ctx 取消正常返回 nil。
func (w *wasapiCapturer) run(ctx context.Context, stream chan<- []int16, errCh chan<- error) error {
	ensureCOM()

	enumIID, err := guidFromString(iidIMMDeviceEnumerator)
	if err != nil {
		return err
	}
	clsid, err := guidFromString(clsidMMDeviceEnumerator)
	if err != nil {
		return err
	}
	clientIID, err := guidFromString(iidIAudioClient)
	if err != nil {
		return err
	}
	captureIID, err := guidFromString(iidIAudioCaptureClient)
	if err != nil {
		return err
	}

	// CoCreateInstance(MMDeviceEnumerator)
	var enumPtr unsafe.Pointer
	hr, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsid)),
		0,
		1, // CLSCTX_INPROC_SERVER
		uintptr(unsafe.Pointer(&enumIID)),
		uintptr(unsafe.Pointer(&enumPtr)),
	)
	if hr != 0 || enumPtr == nil {
		return fmt.Errorf("创建 MMDeviceEnumerator 失败: hr=0x%08x", hr)
	}
	defer releaseCOM(enumPtr)

	dataFlow := uintptr(eCapture)
	if w.source == SourceSystem {
		dataFlow = uintptr(eRender)
	}

	// GetDefaultAudioEndpoint(dataFlow, eConsole)
	var devPtr unsafe.Pointer
	hr = callCOM(enumPtr, vtEnumerator_GetDefaultAudioEndpoint,
		dataFlow, uintptr(eConsole), uintptr(unsafe.Pointer(&devPtr)))
	if hr != 0 || devPtr == nil {
		return fmt.Errorf("获取默认音频设备失败: hr=0x%08x", hr)
	}
	defer releaseCOM(devPtr)

	if audioDebug {
		debugf("设备 ID: %s", deviceID(devPtr))
	}

	// Activate(IAudioClient)
	var clientPtr unsafe.Pointer
	hr = callCOM(devPtr, vtDevice_Activate,
		uintptr(unsafe.Pointer(&clientIID)), 1, 0,
		uintptr(unsafe.Pointer(&clientPtr)))
	if hr != 0 || clientPtr == nil {
		return fmt.Errorf("激活 IAudioClient 失败: hr=0x%08x", hr)
	}
	defer releaseCOM(clientPtr)

	// GetMixFormat
	var mixFmt *waveFormatEx
	hr = callCOM(clientPtr, vtClient_GetMixFormat, uintptr(unsafe.Pointer(&mixFmt)))
	if hr != 0 || mixFmt == nil {
		return fmt.Errorf("获取 MixFormat 失败: hr=0x%08x", hr)
	}
	defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(mixFmt)))

	channels := int(mixFmt.nChannels)
	srcRate := int(mixFmt.nSamplesPerSec)
	isFloat := detectFloat(mixFmt)
	debugf("source=%s mixFormat: rate=%d ch=%d bits=%d tag=0x%04x isFloat=%v",
		w.source, srcRate, channels, mixFmt.wBitsPerSample, mixFmt.wFormatTag, isFloat)

	// Initialize：共享模式、1s 缓冲、轮询驱动。
	// 回环（系统音频）与普通捕获（麦克风）都走轮询，行为一致。
	flags := uintptr(0)
	if w.source == SourceSystem {
		flags = AUDCLNT_STREAMFLAGS_LOOPBACK
	}
	hr = callCOM(clientPtr, vtClient_Initialize,
		uintptr(AUDCLNT_SHAREMODE_SHARED), flags,
		REFTIMES_PER_SEC, 0, uintptr(unsafe.Pointer(mixFmt)), 0)
	if hr != 0 {
		return fmt.Errorf("IAudioClient.Initialize 失败: hr=0x%08x", hr)
	}

	// GetService(IAudioCaptureClient)
	var capturePtr unsafe.Pointer
	hr = callCOM(clientPtr, vtClient_GetService,
		uintptr(unsafe.Pointer(&captureIID)), uintptr(unsafe.Pointer(&capturePtr)))
	if hr != 0 || capturePtr == nil {
		return fmt.Errorf("获取 IAudioCaptureClient 失败: hr=0x%08x", hr)
	}
	defer releaseCOM(capturePtr)

	// Start
	hr = callCOM(clientPtr, vtClient_Start)
	if hr != 0 {
		return fmt.Errorf("IAudioClient.Start 失败: hr=0x%08x", hr)
	}
	defer callCOM(clientPtr, vtClient_Stop)

	conv := NewConverter(srcRate, channels, isFloat)

	// 聚合器：凑满 ChunkSamples 才发一块，保证 100ms 恒定节奏。
	// 回环采集在无声音播放时 WASAPI 不出包，需按经过的时间补零，
	// 让下游（静音计时 / ASR 保活）的节奏不中断。
	var pending []int16
	lastPacket := time.Now()
	flush := func() bool {
		for len(pending) >= ChunkSamples {
			block := make([]int16, ChunkSamples)
			copy(block, pending[:ChunkSamples])
			pending = pending[ChunkSamples:]
			select {
			case stream <- block:
			case <-ctx.Done():
				return false
			}
		}
		return true
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}

		// 超过 50ms 未收到任何包：按空档时长补零
		if gap := time.Since(lastPacket); gap >= 50*time.Millisecond {
			pending = append(pending, make([]int16, int(gap.Seconds()*SampleRate))...)
			lastPacket = time.Now()
			if !flush() {
				return nil
			}
		}

		for {
			var packet uint32
			callCOM(capturePtr, vtCapture_GetNextPacketSize, uintptr(unsafe.Pointer(&packet)))
			if packet == 0 {
				break
			}

			var data *byte
			var frames uint32
			var flagsBuf uint32
			hr := callCOM(capturePtr, vtCapture_GetBuffer,
				uintptr(unsafe.Pointer(&data)),
				uintptr(unsafe.Pointer(&frames)),
				uintptr(unsafe.Pointer(&flagsBuf)), 0, 0)
			if hr != 0 {
				return fmt.Errorf("GetBuffer 失败: hr=0x%08x", hr)
			}

			if frames > 0 {
				if flagsBuf&AUDCLNT_BUFFERFLAGS_SILENT == 0 && data != nil {
					size := int(frames) * channels * w.bytesPerSample(isFloat)
					buf := unsafe.Slice(data, size)
					converted := conv.Convert(buf)
					if audioDebug && len(converted) > 0 {
						debugf("packet: frames=%d converted=%d peakRMS=%.1f", frames, len(converted), RMS(converted))
					}
					pending = append(pending, converted...)
				} else {
					// 静音块：按采样率换算输出样本数补零，维持节奏
					pending = append(pending, make([]int16, int(frames)*SampleRate/srcRate)...)
					debugf("packet: frames=%d (silent)", frames)
				}
				callCOM(capturePtr, vtCapture_ReleaseBuffer, uintptr(frames))
				lastPacket = time.Now()
			}

			if !flush() {
				return nil
			}
			if ctx.Err() != nil {
				return nil
			}
		}
	}
}

func (w *wasapiCapturer) bytesPerSample(isFloat bool) int {
	if isFloat {
		return 4
	}
	return 2
}

// detectFloat 检查 MixFormat 是否为 IEEE float（共享模式通常如此）
func detectFloat(f *waveFormatEx) bool {
	if f.wFormatTag == waveFormatIEEEFloat {
		return true
	}
	if f.wFormatTag == waveFormatExtensible {
		// WAVEFORMATEXTENSIBLE 的 SubFormat 位于 cbSize 之后（偏移 24 字节）
		sub := (*windowsGUID)(unsafe.Pointer(uintptr(unsafe.Pointer(f)) + 24))
		floatGUID, err := guidFromString(subFormatIEEEFloatGUID)
		if err != nil {
			return false
		}
		return *sub == floatGUID
	}
	return false
}

// deviceID 读取 IMMDevice 的 ID 字符串（调试用）
func deviceID(dev unsafe.Pointer) string {
	var pwszID *uint16
	callCOM(dev, vtDevice_GetId, uintptr(unsafe.Pointer(&pwszID)))
	if pwszID == nil {
		return ""
	}
	defer procCoTaskMemFree.Call(uintptr(unsafe.Pointer(pwszID)))
	// ID 是以 NUL 结尾的宽字符；释放前先复制
	var chars []uint16
	for i := 0; i < 512; i++ {
		c := *(*uint16)(unsafe.Add(unsafe.Pointer(pwszID), i*2))
		if c == 0 {
			break
		}
		chars = append(chars, c)
	}
	return string(utf16.Decode(chars))
}

// SystemVolume 读取默认渲染设备的主音量（0~1）与静音状态。
func SystemVolume() (level float64, muted bool, err error) {
	ensureCOM()
	enumIID, err := guidFromString(iidIMMDeviceEnumerator)
	if err != nil {
		return 0, false, err
	}
	clsid, err := guidFromString(clsidMMDeviceEnumerator)
	if err != nil {
		return 0, false, err
	}
	volIID, err := guidFromString(iidIAudioEndpointVolume)
	if err != nil {
		return 0, false, err
	}

	var enumPtr unsafe.Pointer
	hr, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsid)), 0, 1,
		uintptr(unsafe.Pointer(&enumIID)), uintptr(unsafe.Pointer(&enumPtr)),
	)
	if hr != 0 || enumPtr == nil {
		return 0, false, fmt.Errorf("创建 MMDeviceEnumerator 失败: hr=0x%08x", hr)
	}
	defer releaseCOM(enumPtr)

	var devPtr unsafe.Pointer
	hr = callCOM(enumPtr, vtEnumerator_GetDefaultAudioEndpoint,
		uintptr(eRender), uintptr(eConsole), uintptr(unsafe.Pointer(&devPtr)))
	if hr != 0 || devPtr == nil {
		return 0, false, fmt.Errorf("获取默认渲染设备失败: hr=0x%08x", hr)
	}
	defer releaseCOM(devPtr)

	var volPtr unsafe.Pointer
	hr = callCOM(devPtr, vtDevice_Activate,
		uintptr(unsafe.Pointer(&volIID)), 1, 0, uintptr(unsafe.Pointer(&volPtr)))
	if hr != 0 || volPtr == nil {
		return 0, false, fmt.Errorf("激活 IAudioEndpointVolume 失败: hr=0x%08x", hr)
	}
	defer releaseCOM(volPtr)

	var vol float32
	callCOM(volPtr, vtEndpointVol_GetMasterVolumeLevelScalar, uintptr(unsafe.Pointer(&vol)))
	var mute int32
	callCOM(volPtr, vtEndpointVol_GetMute, uintptr(unsafe.Pointer(&mute)))
	return float64(vol), mute != 0, nil
}
