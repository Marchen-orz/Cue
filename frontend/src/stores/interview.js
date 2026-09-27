import { defineStore } from 'pinia'
import { ref } from 'vue'
import { renderMarkdownWithLatex } from '../utils/markdown-latex'

/**
 * 面试模式状态：
 *   transcript — 实时字幕（ASR 中间+句尾结果拼接）
 *   question   — 当前正在回答的问题（触发回答时定格）
 *   answerHtml — 流式答案渲染结果（rAF 节流更新）
 *   error      — 最近一次错误（面板内展示）
 */
export const useInterviewStore = defineStore('interview', () => {
  const isOn = ref(false)
  const isAnswering = ref(false)
  const isThinking = ref(false)
  const transcript = ref('')
  const question = ref('')
  const streamBuffer = ref('')
  const answerHtml = ref('')
  const error = ref(null)

  let rafId = null
  let pendingText = ''

  // rAF 节流的流式渲染（与 solution store 相同的模式）
  function scheduleRender() {
    pendingText = streamBuffer.value
    if (rafId !== null) return
    rafId = requestAnimationFrame(() => {
      rafId = null
      answerHtml.value = renderMarkdownWithLatex(pendingText)
    })
  }

  function handleStreamStart() {
    isAnswering.value = true
    isThinking.value = false
    streamBuffer.value = ''
    answerHtml.value = ''
  }

  function handleStreamChunk(token) {
    isAnswering.value = true
    streamBuffer.value += token
    scheduleRender()
  }

  function handleThinkingChunk() {
    isThinking.value = true
  }

  function handleAnswer(full) {
    isAnswering.value = false
    isThinking.value = false
    streamBuffer.value = full
    answerHtml.value = renderMarkdownWithLatex(full)
  }

  function handleError(msg) {
    error.value = { msg, at: Date.now() }
    isThinking.value = false
    // 保留已收到的部分内容；没有内容才清空回答态
    if (!streamBuffer.value) {
      isAnswering.value = false
    }
  }

  function setState(running) {
    isOn.value = running
    if (!running) {
      isAnswering.value = false
      isThinking.value = false
    }
  }

  function reset() {
    transcript.value = ''
    question.value = ''
    streamBuffer.value = ''
    answerHtml.value = ''
    error.value = null
    isAnswering.value = false
    isThinking.value = false
  }

  return {
    isOn, isAnswering, isThinking,
    transcript, question, streamBuffer, answerHtml, error,
    handleStreamStart, handleStreamChunk, handleThinkingChunk,
    handleAnswer, handleError, setState, reset,
  }
})
