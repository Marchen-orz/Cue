package app

import (
	"Q-Solver/pkg/audio"
	"Q-Solver/pkg/interview"
	"Q-Solver/pkg/logger"
	"Q-Solver/pkg/llm"
)

// StartInterview 启动面试模式（采集 + 语音识别）。
// 需要先配置 ASRAPIKey；自动模式仅在系统音频来源下生效。
func (a *App) StartInterview() {
	defer a.recoverPanics("StartInterview")
	cfg := a.configManager.Get()

	if cfg.ASRAPIKey == "" {
		a.EmitEvent("require-asr-key")
		return
	}

	s := a.getInterviewSession()

	src := audio.Source(cfg.InterviewAudioSource)
	if src != audio.SourceSystem && src != audio.SourceMicrophone {
		src = audio.SourceSystem
	}

	err := s.Start(interview.Config{
		AudioSource: src,
		ASRAPIKey:   cfg.ASRAPIKey,
		ASRModel:    cfg.ASRModel,
		AutoMode:    cfg.InterviewMode != "manual",
		SilenceMs:   cfg.InterviewSilenceMs,
	})
	if err != nil {
		a.EmitEvent("toast", err.Error())
		return
	}
	logger.Println("[interview] 面试模式已启动")
}

// StopInterview 停止面试模式
func (a *App) StopInterview() {
	defer a.recoverPanics("StopInterview")
	if s := a.interviewSession; s != nil {
		s.Stop()
	}
}

// TriggerInterviewAnswer 手动触发回答（用当前累积转写作为问题）
func (a *App) TriggerInterviewAnswer() {
	defer a.recoverPanics("TriggerInterviewAnswer")
	if s := a.interviewSession; s != nil {
		if err := s.TriggerAnswer(); err != nil {
			a.EmitEvent("toast", err.Error())
		}
	}
}

// GetInterviewState 面试模式运行状态
func (a *App) GetInterviewState() bool {
	if s := a.interviewSession; s != nil {
		return s.Running()
	}
	return false
}

// SetInterviewModel 更新面试独立模型（设置修改后调用）
func (a *App) SetInterviewModel(model string) {
	if s := a.interviewSession; s != nil {
		s.SetModelOverride(model)
	}
}

// ==================== 内部 ====================

func (a *App) getInterviewSession() *interview.Session {
	a.interviewMu.Lock()
	defer a.interviewMu.Unlock()
	if a.interviewSession == nil {
		a.interviewSession = interview.NewSession(
			func() llm.Provider { return a.llmService.GetProvider() },
			interview.Events{
				OnState:       a.onInterviewState,
				OnTranscript:  a.onInterviewTranscript,
				OnQuestion:    a.onInterviewQuestion,
				OnStreamStart: func() { a.EmitEvent("interview-stream-start") },
				OnStreamChunk: func(token string) { a.EmitEvent("interview-stream-chunk", token) },
				OnThinking:    func(token string) { a.EmitEvent("interview-stream-thinking", token) },
				OnAnswer:      a.onInterviewAnswer,
				OnError:       a.onInterviewError,
			},
		)
		a.interviewSession.SetResumeFn(func() string {
			return a.configManager.Get().ResumeContent
		})
	}
	return a.interviewSession
}

func (a *App) onInterviewState(running bool) {
	a.EmitEvent("interview-state", running)
}

func (a *App) onInterviewTranscript(text string) {
	a.EmitEvent("interview-transcript", text)
}

func (a *App) onInterviewQuestion(text string) {
	a.lastInterviewQuestion = text
	a.EmitEvent("interview-question", text)
}

func (a *App) onInterviewAnswer(full string) {
	a.EmitEvent("interview-answer", full)
	// 复盘模式：面试问答落盘
	if a.review.IsEnabled() {
		if err := a.review.saveInterviewQA(a.lastInterviewQuestion, full); err != nil {
			logger.Printf("[review] 面试问答保存失败: %v", err)
		}
	}
}

func (a *App) onInterviewError(msg string) {
	a.EmitEvent("interview-error", msg)
}
