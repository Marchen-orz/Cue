import { defineStore } from 'pinia'
import { ref } from 'vue'
import { renderMarkdownWithLatex } from '../utils/markdown-latex'

/**
 * 面试模式状态（对话式一问一答）：
 *   isOn       — 会话存活（含暂停）
 *   isListening — 正在收音（false = 已暂停）
 *   isAnswering — 正在生成回答
 *   transcript  — 实时字幕（ASR 中间+句尾结果拼接）
 *   rounds      — 已完成的问答对 [{question, answer, html}]，流式内容写入最后一轮
 *   error       — 最近一次错误（面板内展示）
 */
export const useInterviewStore = defineStore('interview', () => {
  const isOn = ref(false)
  const isListening = ref(false)
  const isAnswering = ref(false)
  const isThinking = ref(false)
  const transcript = ref('')
  const rounds = ref([]) // { question, answer, html }
  const error = ref(null)

  let rafId = null
  let pendingText = ''

  function lastRound() {
    return rounds.value.length ? rounds.value[rounds.value.length - 1] : null
  }

  // rAF 节流的流式渲染：写入最后一轮的 html
  function scheduleRender() {
    pendingText = streamBuffer()
    if (rafId !== null) return
    rafId = requestAnimationFrame(() => {
      rafId = null
      const last = lastRound()
      if (last) last.html = renderMarkdownWithLatex(pendingText)
    })
  }

  // 当前流式缓冲（挂在 store 局部变量，不进响应式）
  let buffer = ''

  function streamBuffer() {
    return buffer
  }

  function handleStreamStart() {
    isAnswering.value = true
    isThinking.value = false
    buffer = ''
    // 新回答已在 handleQuestion 里建好轮次；没有则兜底建一个
    if (!lastRound() || lastRound().answer) {
      rounds.value.push({ question: '', answer: '', html: '' })
    }
  }

  function handleStreamChunk(token) {
    isAnswering.value = true
    buffer += token
    scheduleRender()
  }

  function handleThinkingChunk() {
    isThinking.value = true
  }

  function handleQuestion(text) {
    rounds.value.push({ question: text, answer: '', html: '' })
  }

  function handleAnswer(full) {
    isAnswering.value = false
    isThinking.value = false
    buffer = full
    const last = lastRound()
    if (last) {
      last.answer = full
      last.html = renderMarkdownWithLatex(full)
    }
  }

  function handleError(msg) {
    error.value = { msg, at: Date.now() }
    isThinking.value = false
    // 保留已收到的部分内容；没有内容才清空回答态
    if (!buffer) {
      isAnswering.value = false
    }
  }

  function setState(running) {
    isOn.value = running
    if (!running) {
      isListening.value = false
      isAnswering.value = false
      isThinking.value = false
    }
  }

  function setListening(listening) {
    isListening.value = !!listening
  }

  // 全新会话：清空对话（暂停/继续不会调用）
  function reset() {
    transcript.value = ''
    rounds.value = []
    buffer = ''
    error.value = null
    isAnswering.value = false
    isThinking.value = false
  }

  return {
    isOn, isListening, isAnswering, isThinking,
    transcript, rounds, error,
    handleStreamStart, handleStreamChunk, handleThinkingChunk,
    handleQuestion, handleAnswer, handleError,
    setState, setListening, reset,
  }
})
