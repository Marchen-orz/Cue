// 假的 Wails 后端：让真实前端脱离 Go 进程在浏览器里跑起来，供截图与宣传片取景。
// 由 serve.mjs 注入到应用页面的 <head>，先于 main.js 执行。
//
// 暴露 window.__cue：
//   emit(event, ...args)   模拟后端事件
//   store(id)              取 Pinia store（ui / settings / solution / interview）
//   freeze()               把页面内所有 CSS 动画/过渡对齐到当前时钟（逐帧渲染用）
//   tick(ms)               ?vclock=1 时把虚拟时钟推进到 ms（逐帧渲染用，见下）
//   script                 舞台逐帧写入的「后端此刻该应答什么」：模型列表、截图预览、已安装的技能
//   pending                挂起中的原生对话框（如导入技能的文件选择框），由舞台在片中「点了打开」时兑现
(() => {
  const params = new URLSearchParams(location.search)

  // ==================== 虚拟时钟 ====================
  // 逐帧渲染时一帧要花上百毫秒真实时间，若让应用跑在真实时钟上，Vue 过渡的超时兜底、
  // toast 定时器都会提前触发。?vclock=1 时接管定时器与 rAF，由舞台页按片内时间推进。
  let tick = async () => {}
  let clockNow = () => performance.now()
  if (params.has('vclock')) {
    let now = 0
    let seq = 1
    const timers = new Map()
    let rafs = []
    const epoch = Date.now()

    window.setTimeout = (fn, delay = 0, ...args) => {
      const id = seq++
      timers.set(id, { at: now + Math.max(0, Number(delay) || 0), fn, args })
      return id
    }
    window.setInterval = (fn, delay = 0, ...args) => {
      const id = seq++
      const every = Math.max(1, Number(delay) || 0)
      timers.set(id, { at: now + every, fn, args, every })
      return id
    }
    window.clearTimeout = window.clearInterval = (id) => { timers.delete(id) }
    window.requestAnimationFrame = (fn) => { const id = seq++; rafs.push({ id, fn }); return id }
    window.cancelAnimationFrame = (id) => { rafs = rafs.filter(r => r.id !== id) }
    performance.now = () => now
    Date.now = () => epoch + now
    clockNow = () => now

    const run = (fn, ...args) => { try { typeof fn === 'function' && fn(...args) } catch (e) { console.error(e) } }

    // 以 60Hz 为步长推进：先触发到期定时器，再跑一轮 rAF，步间让出微任务给 Vue 刷新
    tick = async (target) => {
      while (now < target) {
        const next = Math.min(target, now + 1000 / 60)
        for (;;) {
          let due = null
          for (const [id, tm] of timers) {
            if (tm.at <= next && (!due || tm.at < due.tm.at)) due = { id, tm }
          }
          if (!due) break
          now = Math.max(now, due.tm.at)
          if (due.tm.every) due.tm.at += due.tm.every
          else timers.delete(due.id)
          run(due.tm.fn, ...due.tm.args)
        }
        now = next
        const frame = rafs
        rafs = []
        for (const r of frame) run(r.fn, now)
        await null
        await null
      }
    }
  }

  const listeners = {}

  function on(name, cb, max) {
    const entry = { cb, max: max ?? -1 }
    ;(listeners[name] ||= []).push(entry)
    return () => { listeners[name] = (listeners[name] || []).filter(e => e !== entry) }
  }

  function emit(name, ...args) {
    const list = listeners[name]
    if (!list) return
    for (const e of [...list]) {
      e.cb(...args)
      if (e.max > 0 && --e.max === 0) listeners[name] = listeners[name].filter(x => x !== e)
    }
  }

  // ==================== 假数据 ====================

  const MODELS = [
    'gpt-5.6-sol', 'claude-sonnet-5', 'claude-haiku-4.5', 'claude-opus-5',
    'gpt-5.6-luna', 'gemini-3.8-flash', 'gemini-pro-latest',
    'deepseek-v4-pro', 'deepseek-v4-flash', 'qwen3.8-max-0902', 'qwen3.8-flash',
    'kimi-k3', 'glm-5.3', 'grok-4.6',
  ]

  const SKILLS = [
    {
      id: 'redis-cheatsheet', name: 'Redis 速查', enabled: true, size: 1386,
      description: '缓存穿透 / 击穿 / 雪崩、持久化、分布式锁、过期与淘汰策略：先给结论，再给落地方案。',
      body: '# Redis 速查\n\n- 穿透：布隆过滤器 + 空值缓存\n- 击穿：互斥锁 / 逻辑过期\n- 雪崩：过期时间加随机值 + 多级缓存\n- 分布式锁：`SET key value NX PX`，释放用 Lua 校验持有者',
    },
    {
      id: 'algo-java', name: '算法题解（Java）', enabled: true, size: 1840,
      description: '笔试 / LeetCode 算法题：先给思路与复杂度，再给可直接提交的 Java 代码，最后列出易错边界。',
      body: '# 算法题解（Java）\n\n## 作答顺序\n1. 一句话点明考点（如：滑动窗口 + 哈希表）\n2. 思路：3 步以内说清核心不变量\n3. 复杂度：时间 / 空间\n4. 代码：完整 `class Solution`，可直接提交\n5. 易错点：空串、越界、溢出\n\n## 约束\n- 变量名用完整英文，不写单字母（循环下标除外）\n- 不引入题目未允许的三方库',
    },
    {
      id: 'sql-tuning', name: 'SQL 调优清单', enabled: false, size: 1120,
      description: '慢查询分析与索引设计：执行计划怎么看、联合索引最左前缀、回表与覆盖索引。',
      body: '# SQL 调优清单\n\n- 先看 `EXPLAIN` 的 type / key / rows / Extra\n- 联合索引遵循最左前缀\n- 能覆盖索引就别回表',
    },
    {
      id: 'backend-interview', name: '后端八股速答', enabled: true, size: 3260,
      description: 'Java / MySQL / Redis / 并发 / 网络 常见面试题：按「结论 → 原理 → 追问点」三段作答，口语化、可直接复述。',
      body: '# 后端八股速答\n\n## 回答结构\n- **结论**：一句话先给答案\n- **原理**：2~3 个关键点，带上版本差异（如 JDK 7 / 8）\n- **追问点**：面试官下一句最可能问什么\n\n## 语气\n口语化，像在对面试官讲，不要念文档。',
    },
    {
      id: 'system-design', name: '系统设计框架', enabled: true, size: 2475,
      description: '系统设计类问题的回答框架：需求澄清 → 容量估算 → 接口与数据模型 → 架构 → 瓶颈与取舍。',
      body: '# 系统设计框架\n\n1. 需求澄清：功能 / 非功能\n2. 容量估算：QPS、存储、带宽\n3. 接口与数据模型\n4. 高层架构\n5. 瓶颈与取舍',
    },
  ]

  const config = {
    apiKey: '',
    baseURL: 'https://api.openai.com/v1',
    model: '',
    assistantModel: '',
    prompt: '',
    domainId: 'gen-solver',
    opacity: 1.0,
    noCompression: false,
    compressionQuality: 80,
    sharpening: 0,
    grayscale: true,
    keepContext: false,
    screenshotMode: 'fullscreen',
    resumePath: '',
    resumeContent: '',
    windowWidth: 1024,
    windowHeight: 768,
    screenIndex: 0,
    regionX: 0, regionY: 0, regionW: 0, regionH: 0,
    toolbarCollapsed: false,
    hoverTriggerMs: 1500,
    shieldEnabled: false,
    reviewEnabled: false,
    theme: 'dark',
    interviewAudioSource: 'system',
    interviewMode: 'auto',
    interviewModel: '',
    asrApiKey: '',
    asrModel: 'paraformer-realtime-v2',
    interviewSilenceMs: 800,
    shortcuts: {
      screenshot: { comboID: '119', keyName: 'F8' },
      send: { comboID: '74+162', keyName: 'Ctrl+J' },
      delete: { comboID: '68+162', keyName: 'Ctrl+D' },
      toggle: { comboID: '120', keyName: 'F9' },
      clickthrough: { comboID: '121', keyName: 'F10' },
      toggle_toolbar: { comboID: '118', keyName: 'F7' },
      shield: { comboID: '117', keyName: 'F6' },
      review: { comboID: '115', keyName: 'F4' },
      interview: { comboID: '116', keyName: 'F5' },
      interview_answer: { comboID: '114', keyName: 'F3' },
      move_up: { comboID: '38+164', keyName: 'Alt+↑' },
      move_down: { comboID: '40+164', keyName: 'Alt+↓' },
      move_left: { comboID: '37+164', keyName: 'Alt+←' },
      move_right: { comboID: '39+164', keyName: 'Alt+→' },
      scroll_up: { comboID: '33+164', keyName: 'Alt+PgUp' },
      scroll_down: { comboID: '34+164', keyName: 'Alt+PgDn' },
      opacity_up: { comboID: '38+160+164', keyName: 'Alt+Shift+↑' },
      opacity_down: { comboID: '40+160+164', keyName: 'Alt+Shift+↓' },
    },
  }

  // 让 localStorage 里的主题缓存与假配置一致，避免首帧白→黑闪烁
  const themeParam = params.get('theme')
  if (themeParam) config.theme = themeParam
  try { localStorage.setItem('cue-theme', config.theme) } catch (_) {}

  let domains = null
  const loadDomains = () => (domains ||= fetch('/__promo/domains.json').then(r => r.json()).catch(() => []))

  // 截图预览：画面由舞台页写进 script.preview，缺省给一张占位图
  const placeholder = 'data:image/svg+xml;utf8,' + encodeURIComponent(
    '<svg xmlns="http://www.w3.org/2000/svg" width="640" height="360"><rect width="640" height="360" fill="#f7f8fa"/>' +
    '<rect x="32" y="32" width="300" height="18" rx="4" fill="#1f2937"/><rect x="32" y="70" width="520" height="10" rx="3" fill="#9ca3af"/>' +
    '<rect x="32" y="92" width="470" height="10" rx="3" fill="#9ca3af"/><rect x="32" y="132" width="576" height="96" rx="6" fill="#e5e7eb"/></svg>')

  const state = { shield: false, review: false, stealth: true, clickThrough: false, interview: { running: false, listening: false } }

  const script = {
    models: MODELS,
    preview: { img: placeholder, size: '' },
    skills: SKILLS.map(s => s.id).filter(id => id !== 'redis-cheatsheet'),
  }
  const pending = {}
  const byName = (a, b) => (a.name < b.name ? -1 : a.name > b.name ? 1 : 0)

  // 不碰真实剪贴板；下拉里「滚到已选项」不做平滑滚动（逐帧渲染时没有真实时间可走）
  try {
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText: async () => {}, readText: async () => '' } })
  } catch (_) {}
  const scrollIntoView = Element.prototype.scrollIntoView
  Element.prototype.scrollIntoView = function (o) {
    return scrollIntoView.call(this, o && typeof o === 'object' ? { ...o, behavior: 'auto' } : o)
  }

  // ==================== window.go ====================

  const backend = {
    GetSettings: async () => JSON.parse(JSON.stringify(config)),
    UpdateSettings: async (json) => { try { Object.assign(config, JSON.parse(json)) } catch (_) {} return '' },
    GetDomainCategories: () => loadDomains(),
    GetInitStatus: async () => 'ready',
    GetModels: async () => script.models,
    TestConnection: async () => '',

    GetShieldState: async () => state.shield,
    ToggleShield: async () => { state.shield = !state.shield; emit('shield-state', state.shield) },
    SetShield: async (v) => { state.shield = !!v; emit('shield-state', state.shield) },
    GetReviewState: async () => state.review,
    ToggleReview: async () => { state.review = !state.review; emit('review-state', state.review) },
    SetReview: async (v) => { state.review = !!v; emit('review-state', state.review) },
    GetReviewDir: async () => '',
    SelectReviewDir: async () => '',
    ToggleVisibility: async () => { state.stealth = !state.stealth; emit('toggle-visibility', state.stealth) },
    ToggleClickThrough: async () => { state.clickThrough = !state.clickThrough; emit('click-through-state', state.clickThrough) },

    ListSkills: async () => SKILLS.filter(s => script.skills.includes(s.id)).sort(byName).map(({ body, ...rest }) => ({ ...rest })),
    GetSkill: async (id) => ({ ...SKILLS.find(s => s.id === id) }),
    SetSkillEnabled: async (id, on) => { const s = SKILLS.find(x => x.id === id); if (s) s.enabled = !!on },
    DeleteSkill: async () => {},
    ImportSkillFile: () => new Promise((resolve) => { pending.importSkill = resolve }),
    ImportSkillDir: async () => ({}),

    ListDisplays: async () => [
      { index: 0, x: 0, y: 0, w: 2560, h: 1440, primary: true },
      { index: 1, x: 2560, y: 0, w: 1920, h: 1080, primary: false },
    ],
    // 留一小段「截图中...」，结果取应答那一刻舞台指定的画面
    GetScreenshotPreview: () => new Promise((resolve) => {
      setTimeout(() => resolve({ base64: script.preview.img, size: script.preview.size }), 350)
    }),
    CheckScreenCapturePermission: async () => true,
    TriggerScreenshot: async () => emit('screenshot-taken', script.preview.img, 1),
    SaveImageToFile: async (b64) => { window.__cue.exported = b64; return '' },
    ClearScreenshots: async () => emit('screenshots-cleared'),

    GetInterviewState: async () => ({ ...state.interview }),
    StartInterview: async () => {
      state.interview = { running: true, listening: true }
      emit('interview-state', true); emit('interview-listening', true)
    },
    StopInterview: async () => {
      state.interview = { running: false, listening: false }
      emit('interview-state', false)
    },
    PauseInterview: async () => { state.interview.listening = false; emit('interview-listening', false) },
    ResumeInterview: async () => { state.interview.listening = true; emit('interview-listening', true) },

    CancelRunningTask: async () => true,
  }

  // 未列出的绑定一律返回空 Promise——前端只在个别入口用到返回值
  window.go = { app: { App: new Proxy(backend, { get: (t, k) => t[k] || (async () => undefined) }) } }

  // ==================== window.runtime ====================

  const runtime = {
    EventsOnMultiple: on,
    EventsOff: (...names) => names.forEach(n => { delete listeners[n] }),
    EventsOffAll: () => { for (const k of Object.keys(listeners)) delete listeners[k] },
    EventsEmit: emit,
    WindowGetPosition: async () => ({ x: 0, y: 0 }),
    WindowGetSize: async () => ({ w: innerWidth, h: innerHeight }),
    Environment: async () => ({ buildType: 'dev', platform: 'windows', arch: 'amd64' }),
  }
  window.runtime = new Proxy(runtime, { get: (t, k) => t[k] || (() => Promise.resolve()) })

  // ==================== 取景控制 ====================

  function store(id) {
    const app = document.getElementById('app')?.__vue_app__
    return app?.config.globalProperties.$pinia?._s.get(id)
  }

  // 逐帧渲染时把动画时钟交给舞台：无限循环动画按绝对时间取相位，
  // 一次性动画 / 过渡从首次出现的那一帧起按舞台时间推进。
  // 走完的一次性动画用 finish() 收尾，让 transitionend / animationend 照常触发。
  const seenAt = new WeakMap()
  function freeze() {
    const ms = clockNow()
    for (const a of document.getAnimations()) {
      let start = seenAt.get(a)
      if (start === undefined) { start = ms; seenAt.set(a, start) }
      const timing = a.effect?.getComputedTiming?.() || {}
      if (timing.iterations === Infinity || timing.endTime === Infinity) {
        a.pause()
        a.currentTime = ms
        continue
      }
      const local = Math.max(0, ms - start)
      if (local >= timing.endTime) {
        if (a.playState !== 'finished') a.finish()
      } else {
        a.pause()
        a.currentTime = local
      }
    }
  }

  window.__cue = {
    emit, store, freeze, config, state, skills: SKILLS, models: MODELS, script, pending, exported: '',
    tick: (ms) => tick(ms),
    virtual: params.has('vclock'),
  }
})()
