// Package asr 提供流式语音识别客户端。
// 当前实现：阿里云百炼 Paraformer 实时语音识别（WebSocket 双工协议）。
//
// 协议要点（与官方 Python SDK dashscope-sdk-python 对齐）：
//   - 连接: wss://dashscope.aliyuncs.com/api-ws/v1/inference
//   - 鉴权: 握手 Header "Authorization: Bearer <api-key>"
//   - 指令: run-task / continue-task / finish-task（JSON，header.action 区分）
//   - 音频: continue-task 阶段直接发送二进制 PCM 帧（无需 base64）
//   - 事件: task-started / result-generated / task-finished / task-failed
//   - 结果: payload.output.sentence = {text, begin_time, end_time}，
//     end_time 非空表示句尾（官方 is_sentence_end 判定）
package asr

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// DefaultModel 默认识别模型
const DefaultModel = "paraformer-realtime-v2"

const defaultURL = "wss://dashscope.aliyuncs.com/api-ws/v1/inference"

// 音频参数（与 pkg/audio 输出一致）
const (
	sampleRate = 16000
	// 发送帧大小：4 个 100ms 块（400ms / 12800 字节，与官方 SDK 建议值一致）
	framesPerPacket = 4
)

// Result 一条识别结果
type Result struct {
	Text          string // 当前句文本（句尾前为中间结果，随音频增长）
	IsSentenceEnd bool   // 是否句尾
}

// Client Paraformer 实时识别客户端
type Client struct {
	apiKey string
	model  string
	url    string
}

// NewClient 创建客户端。model 为空用 DefaultModel。
func NewClient(apiKey, model string) *Client {
	if model == "" {
		model = DefaultModel
	}
	return &Client{apiKey: apiKey, model: model, url: defaultURL}
}

// ============ 协议消息结构 ============

type wsHeader struct {
	Action    string `json:"action"`
	TaskID    string `json:"task_id"`
	Streaming string `json:"streaming"`
}

type wsPayload struct {
	TaskGroup string          `json:"task_group,omitempty"`
	Task      string          `json:"task,omitempty"`
	Function  string          `json:"function,omitempty"`
	Model     string          `json:"model,omitempty"`
	Parameters *wsParameters  `json:"parameters,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	Output    *wsOutput       `json:"output,omitempty"`
}

type wsParameters struct {
	Format     string `json:"format"`
	SampleRate int    `json:"sample_rate"`
}

type wsOutput struct {
	Sentence *wsSentence `json:"sentence,omitempty"`
}

type wsSentence struct {
	Text     string `json:"text"`
	BeginTime *int  `json:"begin_time"`
	EndTime  *int   `json:"end_time"`
}

type wsMessage struct {
	Header struct {
		Action       string `json:"action"`
		TaskID       string `json:"task_id"`
		Streaming    string `json:"streaming"`
		Event        string `json:"event"`
		ErrorCode    string `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	} `json:"header"`
	Payload *wsPayload `json:"payload"`
}

// 事件与指令常量
const (
	eventTaskStarted  = "task-started"
	eventResultGen    = "result-generated"
	eventTaskFinished = "task-finished"
	eventTaskFailed   = "task-failed"

	actionRunTask     = "run-task"
	actionFinishTask  = "finish-task"
)

// ============ 识别会话 ============

// Recognize 启动一次流式识别：
//   - audio: 16kHz/16bit/单声道 PCM 块（如 pkg/audio.Capturer 的输出），关闭或 ctx 取消时结束
//   - onResult: 每条中间/句尾结果回调（在识别 goroutine 中调用，需保证并发安全）
//
// 返回前会等待连接关闭。错误返回 task-failed 的 message 或网络错误。
func (c *Client) Recognize(ctx context.Context, audio <-chan []int16, onResult func(Result)) error {
	if c.apiKey == "" {
		return fmt.Errorf("缺少 ASR API Key")
	}

	taskID, err := newTaskID()
	if err != nil {
		return fmt.Errorf("生成 task_id 失败: %w", err)
	}

	dialCtx, cancelDial := context.WithTimeout(ctx, 10*time.Second)
	defer cancelDial()
	conn, resp, err := websocket.DefaultDialer.DialContext(dialCtx, c.url, nil)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("连接 ASR 服务失败: %v (HTTP %d)", err, resp.StatusCode)
		}
		return fmt.Errorf("连接 ASR 服务失败: %w", err)
	}
	defer conn.Close()

	// 读超时：识别静默期可能较长（音频持续发送但服务端可能十几秒无结果）
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// run-task
	startMsg := wsMessage{}
	startMsg.Header.Action = actionRunTask
	startMsg.Header.TaskID = taskID
	startMsg.Header.Streaming = "duplex"
	startMsg.Payload = &wsPayload{
		TaskGroup: "audio",
		Task:      "asr",
		Function:  "recognition",
		Model:     c.model,
		Parameters: &wsParameters{
			Format:     "pcm",
			SampleRate: sampleRate,
		},
		Input: json.RawMessage("{}"),
	}
	if err := conn.WriteJSON(startMsg); err != nil {
		return fmt.Errorf("发送 run-task 失败: %w", err)
	}

	// 等待 task-started
	for {
		var msg wsMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return fmt.Errorf("等待任务启动失败: %w", err)
		}
		if msg.Header.Event == eventTaskStarted {
			break
		}
		if msg.Header.Event == eventTaskFailed {
			return fmt.Errorf("ASR 任务启动失败: %s %s", msg.Header.ErrorCode, msg.Header.ErrorMessage)
		}
	}

	// 双工阶段：发送音频 + 接收结果
	done := make(chan error, 2) // sender/reader 的退出原因
	var once sync.Once
	stop := func(reason error) {
		once.Do(func() { done <- reason })
	}

	go func() { // 发送 goroutine
		defer stop(nil)
		// 发送帧：约 400ms 一包（与官方 SDK 建议 12800 字节对齐）
		pktBytes := make([]byte, 0, sampleRate*2*framesPerPacket/10)
		ticker := time.NewTicker(400 * time.Millisecond)
		defer ticker.Stop()
		flush := func() error {
			if len(pktBytes) == 0 {
				return nil
			}
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.BinaryMessage, pktBytes); err != nil {
				return fmt.Errorf("发送音频帧失败: %w", err)
			}
			pktBytes = pktBytes[:0]
			return nil
		}
		for {
			select {
			case chunk, ok := <-audio:
				if !ok {
					// 音频结束 → finish-task，由 reader 等待 task-finished
					if err := flush(); err != nil {
						stop(err)
						return
					}
					if err := sendFinish(conn, taskID); err != nil {
						stop(err)
						return
					}
					return
				}
				// int16 → little-endian bytes
				for _, v := range chunk {
					pktBytes = append(pktBytes, byte(v&0xff), byte(uint16(v)>>8))
				}
				if len(pktBytes) >= sampleRate*2*framesPerPacket/10 {
					if err := flush(); err != nil {
						stop(err)
						return
					}
				}
			case <-ticker.C:
				// 保活：长时间无音频也周期 flush（服务端 23s 无音频会断开）
				if err := flush(); err != nil {
					stop(err)
					return
				}
			case <-ctx.Done():
				// 通知服务端结束并关闭连接，让 reader 尽快退出
				_ = sendFinish(conn, taskID)
				_ = conn.Close()
				return
			}
		}
	}()

	go func() { // 接收 goroutine
		defer stop(nil)
		for {
			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			var msg wsMessage
			if err := conn.ReadJSON(&msg); err != nil {
				if ctx.Err() != nil || websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return
				}
				// 音频结束后服务端正常关闭也可能以异常帧形式出现
				if strings.Contains(err.Error(), "use of closed network connection") {
					return
				}
				stop(fmt.Errorf("ASR 连接中断: %w", err))
				return
			}
			switch msg.Header.Event {
			case eventResultGen:
				if onResult != nil && msg.Payload != nil && msg.Payload.Output != nil && msg.Payload.Output.Sentence != nil {
					s := msg.Payload.Output.Sentence
					onResult(Result{Text: s.Text, IsSentenceEnd: s.EndTime != nil})
				}
			case eventTaskFailed:
				stop(fmt.Errorf("ASR 识别失败: %s %s", msg.Header.ErrorCode, msg.Header.ErrorMessage))
				return
			case eventTaskFinished:
				return
			}
		}
	}()

	return <-done
}

func sendFinish(conn *websocket.Conn, taskID string) error {
	msg := wsMessage{}
	msg.Header.Action = actionFinishTask
	msg.Header.TaskID = taskID
	msg.Header.Streaming = "duplex"
	msg.Payload = &wsPayload{Input: json.RawMessage("{}")}
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return conn.WriteJSON(msg)
}

func newTaskID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
