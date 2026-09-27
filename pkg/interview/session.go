// Package interview 面试模式会话编排：
//
//	音频采集(16k PCM) → Paraformer 流式识别 → 转写累积与问题检测
//	→ (自动判定/手动触发) → LLM 多轮流式回答 → 前端事件
//
// 并发模型：
//   - stateMu 保护转写缓冲、运行与回答状态
//   - 采集/识别运行在独立 goroutine，结果通过回调进入会话
//   - 回答使用独立的 TaskCoordinator（与搜题任务互不干扰）
package interview

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"Q-Solver/pkg/audio"
	"Q-Solver/pkg/asr"
	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/prompts"
	"Q-Solver/pkg/task"
)

// MaxHistoryRounds 多轮对话保留的最大轮数
const MaxHistoryRounds = 8

// 默认参数
const (
	defaultSilenceMs = 800   // 句尾静音阈值：静默多久判定问题结束
	defaultVoiceRMS  = 300.0 // 16bit PCM 的环境静音 RMS 阈值
	minQuestionRunes = 6     // 触发回答的最小转写长度（字符）
)

// Config 会话启动参数（由 app 层从 config.Config 映射）
type Config struct {
	AudioSource audio.Source // system | mic
	ASRAPIKey   string
	ASRModel    string
	AutoMode    bool    // 自动检测问题并回答（仅系统音频）
	SilenceMs   int     // 自动模式句尾静音阈值
	VoiceRMS    float64 // 有声判定 RMS 阈值
}

// Events 会话事件回调（全部在内部 goroutine 调用，实现方需保证并发安全）
type Events struct {
	OnState       func(running bool)
	OnTranscript  func(text string) // 实时转写字幕（中间结果，会被后续覆盖）
	OnQuestion    func(text string) // 检测到问题（即将回答）
	OnStreamStart func()
	OnStreamChunk func(token string)
	OnThinking    func(token string)
	OnAnswer      func(full string) // 回答完成
	OnError       func(msg string)
}

// Session 面试会话
type Session struct {
	providerFn func() llm.Provider // 每次回答时取当前 Provider
	events     Events
	tasks      *task.TaskCoordinator

	modelMu           sync.Mutex
	modelOverrideName string // 面试独立模型（空 = 主模型）

	resumeMu   sync.Mutex
	resumeFn   func() string
	resumeText string

	stateMu    sync.Mutex
	running    bool
	autoArmed  bool          // 自动模式：检测到待回答的问题
	cancelFn   context.CancelFunc
	transcript strings.Builder // 已完成的转写句子
	partial    string          // 正在识别中的句子
	lastVoiceAt time.Time      // 最后一次有声时刻
	answering  bool

	histMu  sync.Mutex
	history []llm.Message // 多轮对话历史（user/assistant 交替）
}

// NewSession 创建会话。providerFn 在每次回答时调用以获取最新 Provider
// （配置变更后自动生效）。
func NewSession(providerFn func() llm.Provider, events Events) *Session {
	return &Session{
		providerFn: providerFn,
		events:     events,
		tasks:      task.NewTaskCoordinator(),
	}
}

// Running 会话是否正在运行
func (s *Session) Running() bool {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	return s.running
}

// Answering 是否正在生成回答
func (s *Session) Answering() bool {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	return s.answering
}

// Start 启动面试会话（采集 + 识别）。
// 自动模式仅在系统音频下生效；麦克风来源强制退化为手动模式。
func (s *Session) Start(cfg Config) error {
	s.stateMu.Lock()
	if s.running {
		s.stateMu.Unlock()
		return fmt.Errorf("面试模式已在运行")
	}
	auto := cfg.AutoMode
	if auto && cfg.AudioSource != audio.SourceSystem {
		logger.Println("[interview] 自动模式仅支持系统音频，已退化为手动模式")
		auto = false
	}
	if cfg.SilenceMs <= 0 {
		cfg.SilenceMs = defaultSilenceMs
	}
	if cfg.VoiceRMS <= 0 {
		cfg.VoiceRMS = defaultVoiceRMS
	}
	s.running = true
	s.autoArmed = false
	s.transcript.Reset()
	s.partial = ""
	ctx, cancelFn := context.WithCancel(context.Background())
	s.cancelFn = cancelFn
	silence := time.Duration(cfg.SilenceMs) * time.Millisecond
	s.stateMu.Unlock()

	if s.events.OnState != nil {
		s.events.OnState(true)
	}

	go func() {
		defer func() {
			s.stateMu.Lock()
			s.running = false
			s.stateMu.Unlock()
			if s.events.OnState != nil {
				s.events.OnState(false)
			}
		}()

		cap_, err := audio.NewCapturer(cfg.AudioSource)
		if err != nil {
			s.emitError("无法启动音频采集: " + err.Error())
			return
		}
		stream, errCh, err := cap_.Start(ctx)
		if err != nil {
			s.emitError("音频采集启动失败: " + err.Error())
			return
		}

		// 自动模式静音定时器：问题句尾后静默 SilenceMs 触发回答
		silenceTimer := time.AfterFunc(time.Hour, func() {})
		defer silenceTimer.Stop()
		if auto {
			silenceTimer.Stop()
			silenceTimer = time.AfterFunc(time.Hour, func() {
				s.stateMu.Lock()
				armed := s.autoArmed
				answering := s.answering
				voiceRecent := time.Since(s.lastVoiceAt) < silence
				s.stateMu.Unlock()

				if !armed || answering {
					return
				}
				if voiceRecent {
					// 还在说话，继续等
					silenceTimer.Reset(silence)
					return
				}
				s.stateMu.Lock()
				s.autoArmed = false
				text := s.takeTranscriptLocked()
				s.stateMu.Unlock()

				if strings.TrimSpace(stripASRNoise(text)) == "" {
					logger.Println("[interview] 静音触发但转写为空，跳过")
					return
				}
				logger.Println("[interview] 自动触发回答")
				if s.events.OnQuestion != nil {
					s.events.OnQuestion(text)
				}
				s.startAnswer(text)
			})
		}

		// 音频中转：统计有声时刻 + 送 ASR
		asrCh := make(chan []int16, 32)
		go func() {
			defer close(asrCh)
			for {
				select {
				case <-ctx.Done():
					return
				case chunk, ok := <-stream:
					if !ok {
						return
					}
					if audio.RMS(chunk) >= cfg.VoiceRMS {
						s.stateMu.Lock()
						s.lastVoiceAt = time.Now()
						s.stateMu.Unlock()
					}
					select {
					case asrCh <- chunk:
					case <-ctx.Done():
						return
					}
				}
			}
		}()

		onResult := func(r asr.Result) {
			var armed bool
			s.stateMu.Lock()
			if r.IsSentenceEnd {
				s.transcript.WriteString(r.Text)
				s.partial = ""
				runeLen := len([]rune(strings.TrimSpace(r.Text)))
				// 自动模式：句尾 + 未在回答 → 武装静音定时器。
				// 疑问句特征或足够长的陈述都可能是一个问题（是否作答由模型判断，
				// 提示词已要求"没有问题时不要作答"）。
				if auto && !s.answering && runeLen >= minQuestionRunes {
					s.autoArmed = true
					armed = true
					if looksLikeQuestion(r.Text) {
						logger.Printf("[interview] 检测到疑问句尾（%d字）", runeLen)
					} else {
						logger.Printf("[interview] 句尾待判定（%d字，无疑问特征）", runeLen)
					}
				}
			} else {
				s.partial = r.Text
			}
			s.stateMu.Unlock()

			if armed {
				silenceTimer.Reset(silence)
			}
			if s.events.OnTranscript != nil {
				s.events.OnTranscript(s.currentTranscriptText())
			}
		}

		client := asr.NewClient(cfg.ASRAPIKey, cfg.ASRModel)
		err = client.Recognize(ctx, asrCh, onResult)
		if err != nil && ctx.Err() == nil {
			logger.Printf("[interview] ASR 错误: %v", err)
			s.emitError("语音识别中断: " + err.Error())
		}

		// 识别已结束（无论成败）：取消采集上下文，
		// 让采集 goroutine 退出、errCh 关闭，避免会话卡在运行态
		s.stateMu.Lock()
		cancelFn := s.cancelFn
		s.stateMu.Unlock()
		if cancelFn != nil {
			cancelFn()
		}

		// 等待采集 goroutine 退出并收集错误
		for e := range errCh {
			if e != nil && ctx.Err() == nil {
				logger.Printf("[interview] 采集错误: %v", e)
			}
		}
	}()

	return nil
}

// Stop 停止会话（含进行中的回答）
func (s *Session) Stop() {
	s.stateMu.Lock()
	if !s.running {
		s.stateMu.Unlock()
		return
	}
	answering := s.answering
	if s.cancelFn != nil {
		s.cancelFn()
	}
	s.stateMu.Unlock()

	if answering {
		s.tasks.CancelCurrentTask()
	}
}

// TriggerAnswer 手动触发回答：用当前累积的转写作为问题。
// 正在回答时调用会打断当前回答，优先回答新问题。
func (s *Session) TriggerAnswer() error {
	s.stateMu.Lock()
	text := s.takeTranscriptLocked()
	if strings.TrimSpace(stripASRNoise(text)) == "" {
		s.stateMu.Unlock()
		return fmt.Errorf("还没有识别到内容")
	}
	s.autoArmed = false
	if s.answering {
		// 打断当前回答
		if s.tasks.CancelCurrentTask() {
			logger.Println("[interview] 新问题打断当前回答")
		}
	}
	s.stateMu.Unlock()

	if s.events.OnQuestion != nil {
		s.events.OnQuestion(text)
	}
	s.startAnswer(text)
	return nil
}

// ClearHistory 清空多轮对话历史
func (s *Session) ClearHistory() {
	s.histMu.Lock()
	defer s.histMu.Unlock()
	s.history = nil
}

// SetModelOverride 设置面试独立模型（空 = 用主模型）
func (s *Session) SetModelOverride(model string) {
	s.modelMu.Lock()
	s.modelOverrideName = model
	s.modelMu.Unlock()
}

// SetResumeFn 注入简历内容获取函数（每次回答时调用）
func (s *Session) SetResumeFn(fn func() string) {
	s.resumeMu.Lock()
	s.resumeFn = fn
	s.resumeMu.Unlock()
}

// ============ 内部 ============

// currentTranscriptText 当前累积转写（完成句 + 进行中句）
func (s *Session) currentTranscriptText() string {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	return strings.TrimSpace(s.transcript.String() + s.partial)
}

// takeTranscriptLocked 取走当前转写并清空缓冲（须持 stateMu）
func (s *Session) takeTranscriptLocked() string {
	text := strings.TrimSpace(s.transcript.String() + s.partial)
	s.transcript.Reset()
	s.partial = ""
	return text
}

func (s *Session) emitError(msg string) {
	logger.Printf("[interview] %s", msg)
	if s.events.OnError != nil {
		s.events.OnError(msg)
	}
}

// startAnswer 发起回答（独立任务，可被新问题/停止打断）
func (s *Session) startAnswer(question string) {
	provider := s.providerFn()
	if provider == nil {
		s.emitError("模型服务不可用")
		return
	}

	ctx, taskID := s.tasks.StartTask("interview-answer")

	// 回答超时兜底：模型长时间无响应时报错而不是永远卡住
	answerCtx, cancelAnswer := context.WithTimeout(ctx, 2*time.Minute)

	s.stateMu.Lock()
	s.answering = true
	s.stateMu.Unlock()

	go func() {
		start := time.Now()
		logger.Printf("[interview] 开始回答（%d 字）", len([]rune(question)))
		defer func() {
			cancelAnswer()
			s.tasks.CompleteTask(taskID)
			s.stateMu.Lock()
			s.answering = false
			s.stateMu.Unlock()
		}()
		s.answer(answerCtx, provider, question, start)
	}()
}

// answer 执行一次多轮流式回答
func (s *Session) answer(ctx context.Context, provider llm.Provider, question string, start time.Time) {
	messages := s.buildMessages(question)

	if s.events.OnStreamStart != nil {
		s.events.OnStreamStart()
	}

	var full strings.Builder
	chunkCount := 0
	_, err := provider.GenerateContentStreamWithModel(ctx, s.getModelOverride(), messages, func(chunk llm.StreamChunk) {
		switch chunk.Type {
		case llm.ChunkThinking:
			if s.events.OnThinking != nil {
				s.events.OnThinking(chunk.Content)
			}
		case llm.ChunkContent:
			full.WriteString(chunk.Content)
			chunkCount++
			if s.events.OnStreamChunk != nil {
				s.events.OnStreamChunk(chunk.Content)
			}
		}
	})

	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			s.emitError("回答超时（2 分钟无响应），请检查模型服务")
			return
		}
		if ctx.Err() != nil {
			// 被新问题或停止打断
			logger.Printf("[interview] 回答被打断（耗时 %v）", time.Since(start))
			return
		}
		s.emitError("回答失败: " + err.Error())
		return
	}

	answer := full.String()
	if strings.TrimSpace(answer) == "" {
		s.emitError("模型返回内容为空")
		return
	}

	logger.Printf("[interview] 回答完成（%d 字 / %d chunk / 耗时 %v）",
		len([]rune(answer)), chunkCount, time.Since(start))
	s.appendHistory(question, answer)
	if s.events.OnAnswer != nil {
		s.events.OnAnswer(answer)
	}
}

// buildMessages 组装 system + 历史 + 当前问题
func (s *Session) buildMessages(question string) []llm.Message {
	system := prompts.InterviewSystemPrompt

	// 注入简历（与搜题链路的 CandidateProfile 格式一致）
	if rc := s.getResume(); rc != "" {
		system += "\n\n<CandidateProfile>\n  <ResumeSummary>\n" + rc + "\n  </ResumeSummary>\n</CandidateProfile>\n"
	}

	s.histMu.Lock()
	defer s.histMu.Unlock()

	msgs := make([]llm.Message, 0, len(s.history)+2)
	msgs = append(msgs, llm.NewSystemMessage(system))
	msgs = append(msgs, s.history...)
	msgs = append(msgs, llm.NewUserMessage(question))
	return msgs
}

// appendHistory 追加一轮对话并裁剪到 MaxHistoryRounds 轮
func (s *Session) appendHistory(question, answer string) {
	s.histMu.Lock()
	defer s.histMu.Unlock()
	s.history = append(s.history,
		llm.NewUserMessage(question),
		llm.NewAssistantMessage(answer),
	)
	maxMsgs := MaxHistoryRounds * 2
	if len(s.history) > maxMsgs {
		s.history = s.history[len(s.history)-maxMsgs:]
	}
}

func (s *Session) getModelOverride() string {
	s.modelMu.Lock()
	defer s.modelMu.Unlock()
	return s.modelOverrideName
}

func (s *Session) getResume() string {
	s.resumeMu.Lock()
	fn := s.resumeFn
	s.resumeMu.Unlock()
	if fn == nil {
		return ""
	}
	return fn()
}

// ============ 工具 ============

// questionKeywords 疑问句特征（中英文）
var questionKeywords = []string{
	"？", "?",
	"什么", "怎么", "怎样", "如何", "为什么", "哪些", "哪个",
	"多少", "介绍", "讲讲", "说说", "谈谈", "描述", "举例",
	"区别", "对比", "理解", "看法", "优缺点", "能不能", "可不可以",
	"what", "how", "why", "which", "tell", "explain", "describe", "introduce", "difference",
}

// looksLikeQuestion 判断转写文本是否疑似面试官提问
func looksLikeQuestion(text string) bool {
	t := strings.TrimSpace(stripASRNoise(text))
	if len([]rune(t)) < minQuestionRunes {
		return false
	}
	lower := strings.ToLower(t)
	for _, kw := range questionKeywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// stripASRNoise 去除语音识别常见的语气词噪声
func stripASRNoise(text string) string {
	t := strings.TrimSpace(text)
	for _, n := range []string{"嗯", "啊", "呃", "那个", "就是", "好的", "然后"} {
		t = strings.TrimPrefix(t, n)
	}
	return strings.TrimSpace(t)
}
