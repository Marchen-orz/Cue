// Cue 宣传片舞台：render(t) 是 t 的纯函数。
// 桌面、镜头、字幕、系统对话框由本文件直接画；Cue 窗口是 iframe 里的真实前端，
// 每一帧把 appState(t) 同步进它的 Pinia store（见 syncApp），
// store 里没有的界面状态（下拉是否展开、技能开关……）由 settleApp 去点真实的按钮来对齐。
(() => {
  const FPS = 30
  const WARMUP = 4000 // 虚拟时钟预热：跳过应用启动时的「系统就绪」提示

  const APP = { x: 1060, y: 140, w: 840, h: 700 }
  const BAR = { w: 306, h: 50 }
  const SEL = { x0: 22, y0: 152, x1: 700, y1: 800 } // 片中框选的截图区域（题面）
  const IMG = '/__promo/shots/' // 由 render.py --prep 生成的截图素材

  const qs = new URLSearchParams(location.search)
  const CAPTURE = qs.has('capture')
  if (CAPTURE) document.body.classList.add('capture')

  const $ = (s, root = document) => root.querySelector(s)
  const NL = String.fromCharCode(10)
  const clamp = (v, a = 0, b = 1) => Math.min(b, Math.max(a, v))
  const prog = (t, a, b) => clamp((t - a) / (b - a))
  const lerp = (a, b, x) => a + (b - a) * x
  const eo = x => 1 - Math.pow(1 - x, 3)
  const eio = x => (x < 0.5 ? 4 * x * x * x : 1 - Math.pow(-2 * x + 2, 3) / 2)
  const back = x => { const c = 1.70158; return 1 + (c + 1) * Math.pow(x - 1, 3) + c * Math.pow(x - 1, 2) }
  const within = (t, a, b) => t >= a && t < b
  // 出现后保持、再淡出的包络：a 进场、b 退场
  const env = (t, a, b, i = 0.35, o = 0.3) => Math.min(eo(prog(t, a, a + i)), 1 - eo(prog(t, b - o, b)))
  const typed = (text, p) => text.slice(0, Math.round(text.length * clamp(p)))
  const step = (t, pairs) => { let v = pairs[0][1]; for (const [a, x] of pairs) if (t >= a) v = x; return v }
  const count = (t, times) => times.filter(x => x <= t).length
  const series = (a, n, gap) => Array.from({ length: n }, (_, i) => +(a + i * gap).toFixed(3))

  // ==================== 时间轴 ====================
  // 章节起点都落在偶数秒（120 BPM 的小节线）；章内事件写成相对偏移，改一章不牵动其他章。

  const C = {
    bar: 4.6, setup: 8, shots: 34, solve: 42, skills: 58, resume: 68, stealth: 76,
    review: 100, history: 114, look: 124, interview: 136, grid: 174, outro: 178,
  }
  const DUR = 184

  const S = C.setup, H = C.shots, V = C.solve, K = C.skills, R = C.resume, X = C.stealth
  const W = C.review, Y = C.history, L = C.look, I = C.interview

  const T = {
    f7: 5.9,
    // 首次配置 · API
    gear1: S + 0.7, provOpen: S + 1.6, provPick: S + 2.9, keyFocus: S + 3.5, keyA: S + 3.7, keyB: S + 4.7,
    // 首次配置 · 模型与场景
    tabModel: S + 5.4, refresh: S + 6.1, loaded: S + 6.9, modelOpen: S + 7.7, searchA: S + 8.0, searchB: S + 8.6,
    modelPick: S + 9.3, test: S + 10.0, tested: S + 10.9, cat1: S + 11.8, cat2: S + 12.5, domain: S + 13.3,
    // 首次配置 · 截图
    tabShot: S + 14.2, screenOpen: S + 15.3, screenPick: S + 16.0, modeRegion: S + 17.0, regionBtn: S + 18.0,
    dragA: S + 18.9, dragB: S + 20.0, regionDone: S + 20.15, downA: S + 21.3, downB: S + 21.8, gray: S + 22.3,
    qualA: S + 22.9, qualB: S + 23.5, upA: S + 23.8, upB: S + 24.3, save1: S + 25.0,
    // 截图
    shot1: H + 0.8, shot2: H + 2.9, shot3: H + 4.9, del: H + 6.1,
    // 解题
    pgUp: series(V + 10.8, 4, 0.35), copy: V + 13.0, pgDn: series(V + 14.2, 4, 0.35),
    // 技能
    gear2: K + 0.6, tabSkill: K + 1.3, importBtn: K + 2.1, pickZip: K + 3.0, openZip: K + 3.7, sqlOn: K + 5.2, algoOpen: K + 6.4,
    // 简历
    tabResume: R + 0.1, upload: R + 1.0, pickPdf: R + 1.9, openPdf: R + 2.6, parsed: R + 4.8, save2: R + 6.6,
    // 隐身
    opUp: series(X + 5.9, 9, 0.13), f10on: X + 7.6, editor: X + 8.5, typeA: X + 8.7, typeB: X + 10.3,
    hoverA: X + 11.3, hoverFire: X + 12.8, hoverB: X + 14.0, f6: X + 14.6, moveL: X + 16.2, moveR: X + 17.8,
    opDown: series(X + 19.4, 9, 0.13), f10off: X + 21.0,
    // 存储复盘
    f4: W + 0.6, nextQ: W + 1.8, shotQ3: W + 2.9, folderIn: W + 7.9, clickMd: W + 9.2, folderOut: W + 13.3,
    // 外观、历史与导出
    light: Y + 0.9, selA: Y + 2.8, menu: Y + 3.9, export: Y + 4.6, viewerIn: Y + 5.0, viewerOut: Y + 8.5, selB: Y + 9.1,
    // 快捷键与偏好
    gear3: L + 0.8, tabGeneral: L + 1.6, rec: L + 2.6, recorded: L + 3.6, genDownA: L + 4.6, genDownB: L + 5.2,
    hoverDragA: L + 5.9, hoverDragB: L + 6.6, save3: L + 8.6, dark: L + 10.0,
    // 面试
    f5a: I + 1.2, mic: I + 2.6, sys: I + 3.7, silA: I + 4.5, silB: I + 5.1, ivModelOpen: I + 5.8, ivModelPick: I + 6.6,
    asrFocus: I + 7.3, asrA: I + 7.5, asrB: I + 8.5, save4: I + 9.3, f5b: I + 10.8, f3: I + 30.0, pause: I + 35.6,
    // 收尾
    gridA: C.grid, gridB: C.grid + 4.0, outroA: C.outro - 0.1,
  }
  const FLY = 0.62 // 截图缩略图飞进截图坞的时长
  const SA = { send: V + 0.3, think: V + 1.2, answer: V + 3.2, done: V + 10.0, time: '14:30:43' } // 第 2 题
  const SB = { send: W + 4.0, think: null, answer: W + 4.8, done: W + 7.4, time: '14:31:45' } // 第 3 题
  const SAVES = [T.save1, T.save2, T.save3, T.save4]

  // README 静帧取景的时刻（render.py 的 STILLS 用名字引用）
  const MARKS = {
    bar: 5.5, ready: H + 0.4, model: T.domain + 0.7, shotSettings: T.downA - 0.2, solved: SA.done + 0.5,
    skills: T.tabResume - 0.4, split: X + 3.2, through: T.typeB + 0.2, history: T.menu - 0.2,
    shortcuts: T.rec - 0.2, ivSettings: T.save4 - 0.25, interview: I + 26.7, review: T.clickMd + 1.2,
  }

  // ==================== 片中文案 ====================

  const API_KEY = 'sk-or-v1-7f3a9c2e41d84b6a9e05c1d2'
  const ASR_KEY = 'sk-2b9d0c5e8a714f36'
  const MODEL = 'claude-sonnet-5'
  const MODEL_FIRST = 'gpt-5.6-sol' // 刷新后默认选中列表第一项
  const IV_MODEL = 'claude-haiku-4.5'
  const SHOTS = [IMG + 'q2-a.jpg', IMG + 'q2-b.jpg', IMG + 'q2-c.jpg']
  const SHOT3 = IMG + 'q3.jpg'
  const RESUME_PATH = 'D:/求职/后端开发工程师-简历.pdf'
  const REVIEW_DIR = '2026-10-12_143145'

  const THINKING = [
    '题目要求「不含重复字符的最长子串」，子串必须连续 → 典型的滑动窗口。',
    '暴力枚举所有子串是 O(n²)，n 最大 5×10⁴，会超时；进阶也明确要求 O(n)。',
    '用哈希表记录每个字符上次出现的下标：遇到重复字符，直接把左边界跳到它后面，一次遍历即可。',
    '注意左边界只能前进不能回退。',
  ].join(NL)

  const ANSWER = [
    '**考点：滑动窗口 + 哈希表**',
    '',
    '### 思路',
    '1. 用 `left`、`right` 维护一个**不含重复字符**的窗口；',
    '2. `right` 右移时，若当前字符已在窗口内，把 `left` 跳到它上次出现位置的下一位；',
    '3. 每一步用窗口长度更新答案。',
    '',
    '**复杂度**：时间 `O(n)`，空间 `O(k)`（k 为字符集大小）',
    '',
    '### 代码',
    '```java',
    'class Solution {',
    '    public int lengthOfLongestSubstring(String s) {',
    '        Map<Character, Integer> lastSeen = new HashMap<>();',
    '        int best = 0, left = 0;',
    '        for (int right = 0; right < s.length(); right++) {',
    '            char current = s.charAt(right);',
    '            if (lastSeen.containsKey(current)) {',
    '                left = Math.max(left, lastSeen.get(current) + 1);',
    '            }',
    '            lastSeen.put(current, right);',
    '            best = Math.max(best, right - left + 1);',
    '        }',
    '        return best;',
    '    }',
    '}',
    '```',
    '',
    '### 易错点',
    '- `left` 只能前进，别漏了 `Math.max`；',
    '- 空串直接返回 `0`。',
  ].join(NL)

  const ANSWER3 = [
    '**答案：B**',
    '',
    '`volatile` 只保证**可见性**与**有序性**，不保证复合操作的原子性。`count++` 实际是「读 → 加一 → 写」三步：两个线程可能读到同一个旧值，各自加一后写回，丢掉一次更新，所以结果可能小于 2000。',
    '',
    '**排除其他选项**',
    '- A：可见性 ≠ 原子性；',
    '- C：`volatile` 可以修饰任意类型的字段；',
    '- D：只会丢更新，不会多加。',
    '',
    '**改法**：换成 `AtomicInteger.incrementAndGet()`，或给 `increase()` 加 `synchronized`。',
  ].join(NL)

  const RESUME_MD = [
    '# 后端开发工程师 · 2027 届',
    '',
    '## 教育经历',
    '- 计算机科学与技术 · 本科（2023 – 2027），GPA 3.7 / 4.0',
    '',
    '## 项目经历',
    '### 校园二手交易平台 · 秒杀模块（后端负责人）',
    '- Spring Boot + MySQL + Redis + RocketMQ；',
    '- 用 Redis Lua 预扣库存 + 数据库乐观锁解决**库存超卖**，压测 3000 QPS 零超卖；',
    '- 下单链路用 MQ 削峰，接口 P99 从 820 ms 降到 95 ms。',
    '',
    '### 分布式任务调度器',
    '- 基于时间轮实现延时任务，支持失败重试与幂等。',
    '',
    '## 技能',
    'Java · Spring · MySQL · Redis · 消息队列 · Linux',
  ].join(NL)

  const ROUNDS = [
    {
      q: '那我们先聊聊基础，你讲一下 HashMap 的底层实现，还有它是怎么扩容的？',
      a: [
        'HashMap 底层是**数组 + 链表 + 红黑树**（JDK 8 起）。`put` 时先对 key 的 hash 做高低位扰动，再用 `(n - 1) & hash` 定位桶；冲突就挂链表，链表长度超过 8 且数组容量不小于 64 时转成红黑树。',
        '',
        '扩容发生在元素数超过**容量 × 0.75** 时：数组翻倍，每个桶里的节点按 `hash & oldCap` 是 0 还是 1，拆到**原位置**或**原位置 + oldCap**，不需要重新计算 hash。',
      ].join(NL),
      skills: ['后端八股速答'], speak: [I + 11.4, I + 14.6], ask: I + 15.2, ans: [I + 15.6, I + 19.8],
    },
    {
      q: '那它为什么不是线程安全的？并发场景你会用什么？',
      a: '两个线程同时 `put` 会互相覆盖数据，JDK 7 的头插法在并发扩容时还可能成环。并发场景我会用 **ConcurrentHashMap**：JDK 8 用 CAS 加 `synchronized` 只锁单个桶的头节点，粒度比分段锁更细。',
      skills: ['后端八股速答'], speak: [I + 20.4, I + 22.4], ask: I + 23.0, ans: [I + 23.4, I + 26.4],
    },
    {
      q: '我看你简历里做过秒杀模块，库存超卖这个问题，你们当时是怎么解决的？',
      a: [
        '这是我在**校园二手交易平台**里负责的部分，分三层来防：',
        '',
        '1. **Redis 预扣库存**：用 Lua 脚本把「查库存 + 扣减」做成原子操作，挡住绝大部分并发；',
        '2. **数据库兜底**：扣减语句带 `stock > 0` 条件，相当于乐观锁，保证不会扣成负数；',
        '3. **MQ 异步下单**：预扣成功后发消息削峰，下单失败再把库存补回去。',
        '',
        '最后压测 3000 QPS 没有出现超卖，接口 P99 也从 820 ms 降到了 95 ms。',
      ].join(NL),
      skills: ['系统设计框架'], speak: [I + 27.0, I + 29.9], ask: T.f3, ans: [T.f3 + 0.4, I + 34.6],
    },
  ]

  const TYPED_LINE = 'Map<Character, Integer> lastSeen = new HashMap<>();'

  // 左下角字幕条。keys[].at 是按键被按下的时刻（可为数组）
  const LOWER = [
    { a: 4.9, b: S - 0.2, tag: '悬浮窗', title: '一条悬浮工具条，随叫随到', sub: '始终置顶、不抢焦点；一键展开成完整面板',
      keys: [{ combo: ['F7'], label: '展开 / 收起', at: T.f7 }] },
    { a: S + 0.3, b: T.tabModel - 0.2, tag: '01 · 接入模型', title: '用你自己的 API Key', sub: '预设 OpenAI、Google、Anthropic、DeepSeek、阿里云百炼、Moonshot、OpenRouter，也可以填任意兼容接口', keys: [] },
    { a: T.tabModel, b: T.cat1 - 0.4, tag: '01 · 接入模型', title: '选模型，一键测连通', sub: '刷新模型列表、搜索选择，当场验证 Key 与模型可用', keys: [] },
    { a: T.cat1 - 0.2, b: T.tabShot - 0.2, tag: '01 · 接入模型', title: '70 个场景预设', sub: '编程笔试、产设、运营、学术、证书等 7 大类：同一个模型，不同的答法', keys: [] },
    { a: T.tabShot, b: T.regionBtn + 0.1, tag: '02 · 截图设置', title: '截哪里，怎么截，你来定', sub: '整屏、指定显示器，或框选一块固定区域', keys: [] },
    { a: T.regionDone + 0.9, b: T.save1 + 0.8, tag: '02 · 截图设置', title: '区域只框一次', sub: '之后每次截图都直接截这一块；压缩质量、锐化、灰度可调，预览里实时看体积', keys: [] },
    { a: H + 0.2, b: V - 0.1, tag: '03 · 截图解题', title: '看到题，截下来', sub: '区域只框一次，之后按一下就截；最多 3 张合并成一次提问',
      keys: [{ combo: ['F8'], label: '截图', at: [T.shot1, T.shot2, T.shot3] }, { combo: ['Ctrl', 'D'], label: '删除上一张', at: T.del }] },
    { a: V + 0.1, b: V + 5.2, tag: '03 · 截图解题', title: '一键发送，流式作答', sub: '自动匹配技能，思考过程与答案逐字输出',
      keys: [{ combo: ['Ctrl', 'J'], label: '发送解题', at: SA.send }] },
    { a: V + 5.4, b: V + 10.4, tag: '03 · 截图解题', title: '思路、复杂度、可提交的代码', sub: 'Markdown、公式、代码高亮；答题风格由场景预设决定', keys: [] },
    { a: V + 10.6, b: K - 0.2, tag: '03 · 截图解题', title: '免鼠标翻页，一键复制', sub: '看答案不用碰鼠标；代码块与全文都能直接复制',
      keys: [{ combo: ['Alt', 'PgUp'], label: '上翻', at: T.pgUp }, { combo: ['Alt', 'PgDn'], label: '下翻', at: T.pgDn }] },
    { a: K + 0.3, b: T.tabResume - 0.2, tag: '04 · 技能 Skills', title: '给它装上你的技能包', sub: '导入 Claude 格式的 SKILL.md / zip；答题前自动路由，只加载与题目相关的技能', keys: [] },
    { a: T.tabResume, b: X - 0.2, tag: '05 · 简历', title: '让它了解你', sub: '导入 PDF 简历，用你自己的模型整理成 Markdown；回答会贴合你的项目与经历', keys: [] },
    { a: X + 0.7, b: X + 4.6, tag: '06 · 隐身', title: '屏幕共享里，没有它', sub: '对屏幕共享、录屏、截图不可见，你自己照常看得到',
      keys: [{ combo: ['F9'], label: '隐身开关' }] },
    { a: X + 5.5, b: T.hoverA - 0.5, tag: '06 · 隐身', title: '透明、穿透、不抢焦点', sub: '看得见被挡住的内容，鼠标直接点到下层窗口，照常打字',
      keys: [{ combo: ['Alt', 'Shift', '↑'], label: '透明度', at: T.opUp }, { combo: ['F10'], label: '鼠标穿透', at: T.f10on }] },
    { a: T.hoverA - 0.3, b: T.f6 - 0.2, tag: '06 · 隐身', title: '悬停即触发，零点击', sub: '穿透模式下，光标在工具条按钮上停留片刻就触发，全程不产生点击', keys: [] },
    { a: T.f6, b: X + 22.6, tag: '06 · 隐身', title: '快捷键不外泄，窗口随手挪', sub: '快捷键在按下瞬间被拦截，其他程序检测不到；窗口位置、透明度全用键盘调',
      keys: [{ combo: ['F6'], label: '快捷键屏蔽', at: T.f6 }, { combo: ['Alt', '← →'], label: '移动窗口', at: [T.moveL, T.moveR] },
        { combo: ['Alt', 'Shift', '↓'], label: '恢复', at: T.opDown }] },
    { a: W + 0.3, b: T.folderIn - 0.3, tag: '07 · 存储复盘', title: '每道题，自动存档', sub: '开启后，截图与完整解答随答随存',
      keys: [{ combo: ['F4'], label: '存储复盘', at: T.f4 }, { combo: ['F8'], label: '截图', at: T.shotQ3 }, { combo: ['Ctrl', 'J'], label: '发送', at: SB.send }] },
    { a: T.folderIn - 0.1, b: Y - 0.2, tag: '07 · 存储复盘', title: '考后逐题复盘', sub: '每题一份 Markdown，存在 文档/Cue/复盘（目录可改）；面试问答同样归档', keys: [] },
    { a: Y + 0.3, b: T.selA - 0.4, tag: '08 · 外观', title: '浅色 / 深色，一键切换', sub: '两套完整配色', keys: [] },
    { a: T.selA - 0.2, b: L - 0.2, tag: '09 · 历史与导出', title: '历史随时回看，一键导出', sub: '每次解题都留在左侧历史里；任意一条都能导出成一张图片', keys: [] },
    { a: L + 0.3, b: I - 0.5, tag: '10 · 快捷键与偏好', title: '每个快捷键都能重录', sub: '透明度、悬停触发时长、快捷键屏蔽、复盘目录，都在「常规」里',
      keys: [{ combo: ['Alt', 'Q'], label: '录入新按键', at: T.recorded }] },
    { a: I + 1.0, b: T.save4 + 0.9, tag: '11 · 面试模式', title: '先告诉它听什么', sub: '系统音频或麦克风 · 自动 / 手动触发 · 停顿阈值可调 · 可单独指定更快的面试模型',
      keys: [{ combo: ['F5'], label: '面试模式', at: T.f5a }] },
    { a: T.f5b - 0.1, b: I + 20.0, tag: '11 · 面试模式', title: '听题即答', sub: '实时转写面试官的声音，判定为问题、对方停顿后自动作答',
      keys: [{ combo: ['F5'], label: '面试模式', at: T.f5b }] },
    { a: I + 20.2, b: I + 26.7, tag: '11 · 面试模式', title: '连续追问，也接得住', sub: '保留最近 8 轮上下文，追问不用重复背景', keys: [] },
    { a: I + 26.9, b: I + 36.9, tag: '11 · 面试模式', title: '结合你的简历作答', sub: '不想等停顿，按一下立即回答；收音可随时暂停',
      keys: [{ combo: ['F3'], label: '立即回答', at: T.f3 }] },
  ]

  const ICONS = {
    camera: '<path d="M4 8h3l2-3h6l2 3h3v11H4z"/><circle cx="12" cy="13" r="3.5"/>',
    mic: '<rect x="9" y="3" width="6" height="11" rx="3"/><path d="M5 11a7 7 0 0 0 14 0M12 18v3"/>',
    spark: '<path d="M12 3l2 6 6 2-6 2-2 6-2-6-6-2 6-2z"/>',
    eye: '<path d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12z"/><circle cx="12" cy="12" r="3"/><path d="M4 4l16 16"/>',
    archive: '<path d="M3 7h18v3H3z"/><path d="M5 10v9h14v-9"/><path d="M10 14h4"/>',
    file: '<path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z"/><path d="M14 3v5h5"/><path d="M9 13h6M9 17h4"/>',
    layers: '<path d="m12 3 9 5-9 5-9-5z"/><path d="m3 13 9 5 9-5"/>',
    plug: '<path d="M9 3v5M15 3v5"/><path d="M6 8h12v3a6 6 0 0 1-12 0z"/><path d="M12 17v4"/>',
  }
  const CARDS = [
    ['camera', '截图解题', 'F8', '最多 3 张截图合并提问，思路与代码流式输出'],
    ['mic', '面试模式', 'F5', '实时转写，自动判题作答，多轮追问不断线'],
    ['spark', '技能 Skills', '', '导入 Claude 格式技能，按题目自动路由'],
    ['eye', '隐身 · 穿透', 'F9', '对屏幕共享不可见，可透明、可点穿、不抢焦点'],
    ['archive', '存储复盘', 'F4', '截图与解答自动存成 Markdown，考后逐题回看'],
    ['file', '简历解析', '', '导入 PDF 简历，回答贴合你的项目与经历'],
    ['layers', '70 个场景预设', '', '编程、产设、运营、学术、证书……同一个模型，不同的答法'],
    ['plug', '任意兼容接口', '', 'OpenAI、Claude、Gemini、DeepSeek、通义、Kimi、OpenRouter'],
  ]

  // ==================== 桌面 ====================

  const person = (c1, c2) => `<svg viewBox="0 0 100 100"><circle cx="50" cy="40" r="17" fill="${c1}"/><path d="M18 100c0-20 14-32 32-32s32 12 32 32z" fill="${c2}"/></svg>`
  const pre = (...lines) => `<pre>${lines.join(NL)}</pre>`

  const DESK_HTML = `
    <div class="win exam">
      <div class="chrome-tabs">
        <div class="ctab on"><b class="fav"></b><span data-tabtitle>在线笔试 · 编程题 2 / 3</span><span class="x">✕</span></div>
        <div class="ctab"><b class="fav g"></b>笔试须知</div>
        <div class="wctl"><i>—</i><i>▢</i><i>✕</i></div>
      </div>
      <div class="chrome-bar">
        <span class="nav"><i></i><i></i><i></i></span>
        <div class="url"><span class="lock"></span><span data-url>exam.campus-hire.example/session/8f2a/problem/2</span></div>
        <span class="rec"><i></i>正在共享你的屏幕</span>
      </div>
      <div class="page">
        <div class="xhead">
          <div class="xbrand"><b></b>2027 届校园招聘 · 后端开发工程师笔试</div>
          <div class="xsteps"><span>1</span><span>2</span><span>3</span></div>
          <div class="xright">
            <span class="mon"><i></i>监考中 · 屏幕与摄像头已开启</span>
            <span class="timer">剩余<b data-timer>01:12:36</b></span>
            <span class="submit">交卷</span>
          </div>
        </div>
        <div class="xbody">
          <div class="xprob">
            <div class="xq" data-q="2"><div class="xscroll" data-scroll>
              <span class="ptag">第 2 题 · 编程题 · 30 分</span>
              <h2>无重复字符的最长子串</h2>
              <p>给定一个字符串 <code>s</code>，请你找出其中<b>不含有重复字符</b>的<b>最长子串</b>的长度。</p>
              <h4>示例 1</h4>
              ${pre('输入：s = "abcabcbb"', '输出：3', '解释：无重复字符的最长子串是 "abc"，长度为 3。')}
              <h4>示例 2</h4>
              ${pre('输入：s = "bbbbb"', '输出：1')}
              <h4>示例 3</h4>
              ${pre('输入：s = "pwwkew"', '输出：3', '解释：答案是 "wke"；注意 "pwke" 是子序列而不是子串。')}
              <h4>数据范围</h4>
              <ul><li>0 ≤ s.length ≤ 5 × 10⁴</li><li>s 由英文字母、数字、符号和空格组成</li></ul>
              <h4>限制</h4>
              <p>时间限制 1 s · 内存限制 256 MB · 语言 Java 17</p>
              <h4>输入格式</h4>
              <p>一行，一个字符串 <code>s</code>。<code>s</code> 可能为空串，也可能包含空格，读入时请不要丢掉首尾空白。</p>
              <h4>输出格式</h4>
              <p>一个整数，表示不含重复字符的最长子串的长度。</p>
              <h4>样例说明</h4>
              <table>
                <tr><th>输入</th><th>输出</th><th>最长子串</th></tr>
                <tr><td><code>"abcabcbb"</code></td><td>3</td><td><code>"abc"</code></td></tr>
                <tr><td><code>"bbbbb"</code></td><td>1</td><td><code>"b"</code></td></tr>
                <tr><td><code>"pwwkew"</code></td><td>3</td><td><code>"wke"</code></td></tr>
                <tr><td><code>""</code></td><td>0</td><td>（空）</td></tr>
                <tr><td><code>"dvdf"</code></td><td>3</td><td><code>"vdf"</code></td></tr>
              </table>
              <h4>进阶</h4>
              <p>你能给出时间复杂度为 <code>O(n)</code> 的解法吗？要求只遍历一次字符串，额外空间与字符集大小成正比。</p>
              <h4>提示</h4>
              <ul><li>子串必须连续，子序列不要求连续；</li><li>注意空串以及只含空格的输入；</li><li>测试数据中包含长度为 5 × 10⁴ 的极端用例，O(n²) 的解法会超时。</li></ul>
              <h4>评分说明</h4>
              <ul><li>共 20 组测试数据，按通过的比例给分；</li><li>超时或内存超限的测试点不得分；</li><li>可以多次提交，以最后一次提交为准。</li></ul>
              <h4>注意事项</h4>
              <p>请不要修改类名与方法签名；本题不允许使用题目之外的第三方库。作答过程全程录屏，切出考试页面将被记录。</p>
            </div><i class="xthumb" data-thumb></i></div>
            <div class="xq" data-q="3">
              <span class="ptag">第 3 题 · 单选题 · 10 分</span>
              <h2>volatile 与线程安全</h2>
              <p>两个线程各调用 1000 次 <code>increase()</code>，全部结束后再读取 <code>count</code>。关于下面这段代码，说法<b>正确</b>的是：</p>
              ${pre('class Counter {', '    private volatile int count = 0;', '', '    public void increase() { count++; }', '    public int get() { return count; }', '}')}
              <div class="opts">
                <div class="opt"><b>A</b><span>结果一定是 2000，<code>volatile</code> 已经保证了线程安全</span></div>
                <div class="opt"><b>B</b><span>结果可能小于 2000，<code>volatile</code> 不保证 <code>count++</code> 的原子性</span></div>
                <div class="opt"><b>C</b><span>编译失败，<code>volatile</code> 不能修饰基本类型</span></div>
                <div class="opt"><b>D</b><span>结果可能大于 2000，因为发生了指令重排</span></div>
              </div>
            </div>
          </div>
          <div class="xedit" data-edit>
            <div class="etabs"><span class="lang">Java 17</span><span>Solution.java</span><span class="run">▶ 运行测试</span></div>
            <div class="code">
              <div class="ln"><span class="n">1</span><span><span class="k">import</span> java.util.*;</span></div>
              <div class="ln"><span class="n">2</span></div>
              <div class="ln"><span class="n">3</span><span><span class="k">class</span> <span class="t">Solution</span> {</span></div>
              <div class="ln"><span class="n">4</span><span>    <span class="k">public int</span> <span class="f">lengthOfLongestSubstring</span>(<span class="t">String</span> s) {</span></div>
              <div class="ln hl"><span class="n">5</span><span>        <span data-typed></span><span class="caret" data-caret></span></span></div>
              <div class="ln"><span class="n">6</span><span>        <span class="k">return</span> <span class="m">0</span>;</span></div>
              <div class="ln"><span class="n">7</span><span>    }</span></div>
              <div class="ln"><span class="n">8</span><span>}</span></div>
            </div>
            <div class="console">
              <div class="ct"><b>测试用例</b><span>运行结果</span><span>提交记录</span></div>
              ${pre('s = "abcabcbb"', '', '尚未运行。点击「运行测试」查看输出。')}
            </div>
          </div>
          <div class="xsheet" data-sheet>
            <h3>答题卡</h3>
            <p>本题为单选题，请在左侧点击选项作答；交卷前可以随时修改。</p>
            <div class="sheet-grid"><span class="ok">1</span><span class="ok">2</span><span class="now">3</span></div>
            <div class="sheet-note">第 1、2 题已保存 · 第 3 题尚未作答</div>
          </div>
        </div>
      </div>
    </div>
    <div class="win meet">
      <div class="mhead"><i class="live"></i>视频面试 · 后端开发工程师 · 技术一面
        <span class="mshare"><i></i>你正在共享屏幕</span><span class="mtime" data-mtime>12:48</span></div>
      <div class="tile big" data-speaker>
        <div class="glow"></div><div class="ring" data-ring></div>
        <div class="avatar">${person('#94a3b8', '#64748b')}</div>
        <div class="nametag"><span class="eq" data-eq><i></i><i></i><i></i><i></i><i></i></span>面试官 · 陈工</div>
      </div>
      <div class="tile t2"><div class="avatar">${person('#a1a1aa', '#71717a')}</div><div class="nametag">HR · 林</div></div>
      <div class="tile t3"><div class="avatar">${person('#7dd3fc', '#0ea5e9')}</div><div class="nametag">我</div></div>
      <div class="mbar"><div class="mbtn"><i></i></div><div class="mbtn"><i></i></div><div class="mbtn on"><i></i></div><div class="mbtn"><i></i></div><div class="mbtn end"><i></i></div></div>
    </div>
    <div class="taskbar"><i class="a"></i><i></i><i class="b"></i><i class="c"></i><i></i><i></i>
      <div class="clock"><span data-clock>14:30</span><br>2026/10/12</div></div>`

  const desks = [...document.querySelectorAll('.desk')]
  for (const d of desks) d.innerHTML = DESK_HTML

  const pad = n => String(n).padStart(2, '0')

  function updateDesk(root, t, o) {
    const exam = $('.exam', root), meet = $('.meet', root)
    exam.style.opacity = o.meet ? 0 : 1
    meet.style.opacity = o.meet ? 1 : 0
    const left = 4356 - Math.floor(t)
    $('[data-timer]', root).textContent = `${pad(Math.floor(left / 3600))}:${pad(Math.floor(left / 60) % 60)}:${pad(left % 60)}`
    $('[data-clock]', root).textContent = `14:${pad(30 + Math.floor(Math.max(0, t) / 60))}`
    $('[data-typed]', root).textContent = o.typed
    $('[data-caret]', root).style.opacity = o.caret && Math.floor(t * 2) % 2 === 0 ? 1 : 0

    // 第 2 题（可滚动的编程题）与第 3 题（单选题）
    const q3 = o.q === 3
    if (root.dataset.q !== String(o.q)) {
      root.dataset.q = o.q
      $('[data-tabtitle]', root).textContent = q3 ? '在线笔试 · 单选题 3 / 3' : '在线笔试 · 编程题 2 / 3'
      $('[data-url]', root).textContent = `exam.campus-hire.example/session/8f2a/problem/${o.q}`
      ;[...$('.xsteps', root).children].forEach((n, i) => { n.className = i + 1 < o.q ? 'done' : i + 1 === o.q ? 'cur' : '' })
      $('[data-q="2"]', root).style.display = q3 ? 'none' : ''
      $('[data-q="3"]', root).style.display = q3 ? '' : 'none'
      $('[data-edit]', root).style.display = q3 ? 'none' : ''
      $('[data-sheet]', root).style.display = q3 ? '' : 'none'
    }
    const scroll = $('[data-scroll]', root), thumb = $('[data-thumb]', root)
    const view = scroll.parentElement.clientHeight, full = scroll.scrollHeight
    const max = Math.max(0, full - view)
    scroll.style.transform = `translateY(${-max * clamp(o.scroll)}px)`
    const th = Math.max(40, view * view / Math.max(full, 1))
    Object.assign(thumb.style, { height: th + 'px', top: 6 + (view - 12 - th) * clamp(o.scroll) + 'px' })

    const sec = 768 + Math.floor(Math.max(0, t - I))
    $('[data-mtime]', root).textContent = `${pad(Math.floor(sec / 60))}:${pad(sec % 60)}`
    $('[data-speaker]', root).style.borderColor = `rgba(34,197,94,${0.85 * o.speaking})`
    const ring = $('[data-ring]', root)
    const beat = (t * 1.6) % 1
    ring.style.opacity = o.speaking * (1 - beat) * 0.7
    ring.style.transform = `scale(${1 + beat * 0.28})`
    ;[...$('[data-eq]', root).children].forEach((bar, i) => {
      const h = 4 + o.speaking * 12 * Math.abs(Math.sin(t * (7 + i * 1.7) + i * 1.3))
      bar.style.height = h + 'px'
    })
  }

  // ==================== 系统窗口（文件选择框 / 下拉列表 / 资源管理器 / 图片查看器） ====================

  const DIALOGS = {
    skill: {
      path: ['此电脑', '下载', 'cue-skills'], filter: '技能文件 (*.md; *.zip)', pick: 1,
      rows: [
        ['dir', 'system-design', '2026/10/08 21:14', '文件夹', ''],
        ['zip', 'redis-速查.zip', '2026/10/11 22:40', '压缩文件夹', '4 KB'],
        ['md', 'sql-调优清单.md', '2026/10/09 19:02', 'Markdown 文档', '2 KB'],
        ['md', '行为面试-STAR.md', '2026/09/27 10:36', 'Markdown 文档', '3 KB'],
      ],
    },
    resume: {
      path: ['此电脑', '文档', '求职'], filter: 'PDF 文件 (*.pdf)', pick: 1,
      rows: [
        ['dir', '作品集', '2026/09/30 18:20', '文件夹', ''],
        ['pdf', '后端开发工程师-简历.pdf', '2026/10/10 16:25', 'PDF 文档', '212 KB'],
        ['pdf', '成绩单.pdf', '2026/09/12 09:41', 'PDF 文档', '488 KB'],
      ],
    },
  }
  const crumbs = (parts) => parts.map(p => `<span>${p}</span>`).join('<b>›</b>')
  const SIDE = ['桌面', '下载', '文档', '图片'].map(n => `<div data-side="${n}"><i></i>${n}</div>`).join('')
  const fileRow = ([kind, name, date, type, size], cls = '') =>
    `<div class="frow ${cls}"><span><i class="${kind}"></i>${name}</span><span class="dim">${date}</span><span class="dim">${type}</span><span class="dim">${size}</span></div>`
  const FHEAD = '<div class="fhead"><span>名称</span><span>修改日期</span><span>类型</span><span>大小</span></div>'
  const WCTL = '<div class="ctl"><i>—</i><i>▢</i><i>✕</i></div>'

  const fdlg = $('#fdlg')
  let fdlgKind = null
  function fillDialog(kind) {
    if (fdlgKind === kind) return
    fdlgKind = kind
    const d = DIALOGS[kind]
    fdlg.innerHTML = `
      <div class="tbar"><b class="ico"></b>打开<div class="ctl"><i>✕</i></div></div>
      <div class="addr"><span class="navs">←→↑</span><div class="path">${crumbs(d.path)}</div><div class="find">搜索</div></div>
      <div class="osbody"><div class="oside">${SIDE}</div>
        <div class="flist">${FHEAD}${d.rows.map((r, i) => fileRow(r, i === d.pick ? 'pick' : '')).join('')}</div></div>
      <div class="ffoot"><span>文件名(N)：</span><div class="inp" data-name></div><div class="inp sel">${d.filter}<em>▾</em></div>
        <div class="btns"><div class="obtn pri">打开(O)</div><div class="obtn">取消</div></div></div>`
    $(`[data-side="${d.path[1]}"]`, fdlg).classList.add('on')
  }

  $('#selpop').innerHTML = '<div>屏幕 1（2560×1440，主屏）</div><div>屏幕 2（1920×1080）</div>'

  $('#explorer').innerHTML = `
    <div class="tbar"><b class="ico folder"></b>${REVIEW_DIR}${WCTL}</div>
    <div class="addr"><span class="navs">←→↑</span><div class="path">${crumbs(['文档', 'Cue', '复盘', REVIEW_DIR])}</div><div class="find">搜索</div></div>
    <div class="osbody">
      <div class="flist">${FHEAD}
        ${fileRow(['dir', 'images', '2026/10/12 14:31', '文件夹', ''])}
        ${fileRow(['md', '题目01.md', '2026/10/12 14:31', 'Markdown 文档', '1 KB'], 'mdrow')}
        <div class="fnote">每开启一次复盘新建一个会话文件夹；面试问答存为「面试问答01.md」。</div>
      </div>
      <div class="pview" data-pview>
        <div class="pv-empty">选择要预览的文件</div>
        <div class="pv-md">
          <h1>题目01</h1>
          <blockquote>时间：2026-10-12 14:31:48</blockquote>
          <h2>题目截图</h2>
          <img src="${SHOT3}" alt="">
          <h2>解答</h2>
          <p><b>答案：B</b></p>
          <p><code>volatile</code> 只保证<b>可见性</b>与<b>有序性</b>，不保证复合操作的原子性。<code>count++</code> 实际是「读 → 加一 → 写」三步……</p>
        </div>
      </div>
    </div>`

  $('#viewer').innerHTML = `
    <div class="tbar"><b class="ico photo"></b>cue-export.png - 照片${WCTL}</div>
    <div class="vbody"><div class="vimg" data-vimg style="background-image:url(${IMG}export.png)"></div></div>
    <div class="vfoot"><span>已保存到 图片 / cue-export.png</span><span>100%</span></div>`

  // ==================== 片头 / 片尾 / 宫格 ====================

  const logoSVG = (id) => `
    <svg class="logo" viewBox="0 0 1024 1024">
      <defs>
        <linearGradient id="bg${id}" x1="140" y1="64" x2="884" y2="960" gradientUnits="userSpaceOnUse"><stop offset="0" stop-color="#4cc3fb"/><stop offset="1" stop-color="#0a84d0"/></linearGradient>
        <radialGradient id="gl${id}" cx="300" cy="190" r="620" gradientUnits="userSpaceOnUse"><stop offset="0" stop-color="#fff" stop-opacity=".28"/><stop offset="1" stop-color="#fff" stop-opacity="0"/></radialGradient>
      </defs>
      <rect x="64" y="64" width="896" height="896" rx="216" fill="url(#bg${id})"/>
      <rect x="64" y="64" width="896" height="896" rx="216" fill="url(#gl${id})"/>
      <path data-arc d="M598 365 A196 196 0 1 0 598 659" fill="none" stroke="#fff" stroke-width="92" stroke-linecap="round" stroke-dasharray="830"/>
      <circle data-dot cx="716" cy="512" r="52" fill="#d7f0ff" style="transform-box: fill-box; transform-origin: center"/>
    </svg>`

  const REPO = qs.get('repo') || 'github.com/Marchen-orz/Cue'
  const GH = '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2a10 10 0 0 0-3.2 19.5c.5.1.7-.2.7-.5v-1.7c-2.8.6-3.4-1.2-3.4-1.2-.5-1.2-1.1-1.5-1.1-1.5-.9-.6.1-.6.1-.6 1 .1 1.5 1 1.5 1 .9 1.5 2.4 1.1 2.9.8.1-.6.4-1.1.6-1.3-2.2-.3-4.6-1.1-4.6-5a3.9 3.9 0 0 1 1-2.7c-.1-.3-.4-1.3.1-2.7 0 0 .8-.3 2.8 1a9.500 9.500 0 0 1 5 0c1.900-1.300 2.700-1 2.700-1 .500 1.400.200 2.400.100 2.700a3.900 3.900 0 0 1 1 2.700c0 3.900-2.400 4.700-4.600 5 .400.300.700.900.700 1.800V21c0 .300.200.600.700.500A10 10 0 0 0 12 2z"/></svg>'

  $('#intro').innerHTML = `<div class="hero">
      <div class="brand">${logoSVG('i')}<div class="word" data-word>&nbsp;Cue</div></div>
      <div class="tagline" data-tag>截图即问<b>·</b>听题即答<b>·</b>悬浮隐身</div>
      <div class="subline" data-sub>开源桌面 AI 助手 · Windows / macOS</div>
      <div class="ping" data-ping></div>
    </div>`

  const chip = (keys, label) => `<div class="chip">${keys.map(k => `<kbd>${k}</kbd>`).join('<span class="plus">+</span>')}${label}</div>`
  $('#outro').innerHTML = `<div class="hero">
      <div class="brand">${logoSVG('o')}<div class="word">&nbsp;Cue</div></div>
      <div class="tagline" data-tag>截图即问<b>·</b>听题即答<b>·</b>悬浮隐身</div>
      <div class="chips" data-chips>${chip(['F8'], '截图')}${chip(['Ctrl', 'J'], '解题')}${chip(['F5'], '面试模式')}${chip(['F9'], '隐身')}</div>
      <div class="repo" data-repo>${GH}${REPO}</div>
      <div class="credit" data-credit>基于 Q-Solver 二次开发 · CC BY-NC 4.0 · 仅供学习与练习使用</div>
    </div>`

  $('#more .cards').innerHTML = CARDS.map(([, title, key, desc], i) =>
    `<div class="card"><div class="n">${String(i + 1).padStart(2, '0')}</div><h3>${title}${key ? `<kbd>${key}</kbd>` : ''}</h3><p>${desc}</p></div>`).join('')

  function renderIntro(t) {
    const el = $('#intro')
    const out = eio(prog(t, 3.7, 4.4))
    el.style.display = t < 4.4 ? '' : 'none'
    if (t >= 4.4) return
    el.style.opacity = 1 - out
    const hero = $('.hero', el)
    hero.style.transform = `scale(${1 + out * 0.35})`
    const logo = $('.logo', el)
    const pop = prog(t, 0.2, 0.85)
    logo.style.opacity = eo(prog(t, 0.2, 0.5))
    logo.style.transform = `scale(${0.6 + 0.4 * back(pop)})`
    const arc = prog(t, 0.55, 1.3)
    const path = $('[data-arc]', el)
    path.style.strokeDashoffset = 830 * (1 - eio(arc))
    path.style.opacity = arc > 0.001 ? 1 : 0
    const dot = prog(t, 1.3, 1.6)
    $('[data-dot]', el).style.transform = `scale(${dot <= 0 ? 0 : back(dot)})`
    // 圆点「发声」的涟漪，圆心对齐 logo 内的圆点
    const ping = $('[data-ping]', el)
    const pp = prog(t, 1.32, 2.2)
    const lr = logo.getBoundingClientRect(), hr = hero.getBoundingClientRect()
    const k = hr.width / hero.offsetWidth || 1
    const cx = (lr.left - hr.left + lr.width * (716 / 1024)) / k
    const cy = (lr.top - hr.top + lr.height * 0.5) / k
    const size = 20 + eo(pp) * 300
    Object.assign(ping.style, { left: cx - size / 2 + 'px', top: cy - size / 2 + 'px', width: size + 'px', height: size + 'px', opacity: pp > 0 ? (1 - pp) * 0.9 : 0 })
    const word = $('[data-word]', el)
    const w = eo(prog(t, 1.45, 2.1))
    word.style.maxWidth = w * 420 + 'px'
    word.style.opacity = w
    for (const [sel, a] of [['[data-tag]', 2.0], ['[data-sub]', 2.4]]) {
      const p = eo(prog(t, a, a + 0.6))
      const node = $(sel, el)
      node.style.opacity = p
      node.style.transform = `translateY(${(1 - p) * 22}px)`
    }
  }

  function renderOutro(t) {
    const el = $('#outro')
    const a = T.outroA
    el.style.display = t >= a ? '' : 'none'
    if (t < a) return
    el.style.opacity = eio(prog(t, a, a + 0.7)) * (1 - eio(prog(t, DUR - 0.9, DUR - 0.1)))
    const path = $('[data-arc]', el)
    path.style.strokeDashoffset = 830 * (1 - eio(prog(t, a + 0.3, a + 1.0)))
    $('[data-dot]', el).style.transform = `scale(${back(prog(t, a + 1.0, a + 1.3)) * (t >= a + 1.0 ? 1 : 0)})`
    $('.hero', el).style.transform = `scale(${0.94 + 0.06 * eo(prog(t, a, a + 1.4))})`
    const rise = [['[data-tag]', 0.7], ['[data-chips]', 1.1], ['[data-repo]', 1.5], ['[data-credit]', 1.9]]
    for (const [sel, d] of rise) {
      const p = eo(prog(t, a + d, a + d + 0.6))
      const node = $(sel, el)
      node.style.opacity = p
      node.style.transform = `translateY(${(1 - p) * 22}px)`
    }
  }

  function renderMore(t) {
    const el = $('#more')
    const a = T.gridA, b = T.gridB
    const on = within(t, a, b + 0.1)
    el.style.display = on ? '' : 'none'
    if (!on) return
    el.style.opacity = env(t, a, b + 0.1, 0.45, 0.4)
    const h = eo(prog(t, a + 0.1, a + 0.6))
    Object.assign($('.more-h', el).style, { opacity: h, transform: `translateY(${(1 - h) * 20}px)` })
    ;[...el.querySelectorAll('.card')].forEach((c, i) => {
      const p = prog(t, a + 0.35 + i * 0.09, a + 0.85 + i * 0.09)
      c.style.opacity = eo(prog(t, a + 0.35 + i * 0.09 + 0.15, a + 0.75 + i * 0.09))
      c.style.setProperty('--rule', eio(p))
      c.style.transform = `translateY(${(1 - eo(p)) * 24}px)`
    })
  }

  // ==================== 镜头 ====================
  // [tx, ty, scale]：世界左上角在画面中的位置与缩放

  const FULL = [0, 0, 1]
  const TOOLBAR = [-1344, -102, 1.7]
  const PANEL = [-576, -117, 1.3]
  const ZOOM = [-864, -171, 1.45]
  const SPLIT = [44, 205, 0.47]
  const MID = [-384, -72, 1.2]
  const CAMERA = [
    [0, TOOLBAR], [T.f7, TOOLBAR], [T.f7 + 0.7, PANEL], [S + 0.1, PANEL], [S + 0.9, ZOOM],
    [T.regionBtn + 0.05, ZOOM], [T.regionBtn + 0.75, FULL], [T.regionDone + 0.2, FULL], [T.regionDone + 1.0, ZOOM],
    [T.save1 + 0.2, ZOOM], [T.save1 + 1.0, FULL],
    [SA.send - 0.1, FULL], [SA.send + 0.8, ZOOM],
    [X, ZOOM], [X + 0.9, SPLIT], [X + 4.6, SPLIT], [X + 5.5, MID],
    [X + 22.4, MID], [X + 23.3, FULL],
    [Y, FULL], [Y + 0.8, ZOOM],
    [I - 0.02, ZOOM], [I, MID], [T.f5a + 0.1, MID], [T.f5a + 0.9, ZOOM],
    [T.save4 + 0.2, ZOOM], [T.save4 + 1.0, MID], [DUR, MID],
  ]

  function track(t, kf) {
    if (t <= kf[0][0]) return kf[0][1]
    for (let i = 1; i < kf.length; i++) {
      if (t <= kf[i][0]) {
        const p = eio(prog(t, kf[i - 1][0], kf[i][0]))
        return kf[i][1].map((v, j) => lerp(kf[i - 1][1][j], v, p))
      }
    }
    return kf[kf.length - 1][1]
  }

  // ==================== 光标 ====================

  const frame = $('#app')
  const world = $('#world'), share = $('#share'), wrap = $('#appwrap')
  let cue = null
  let pos = { x: APP.x, y: APP.y } // Cue 窗口此刻的位置（Alt+方向键会挪动它）

  // 落点：应用内的元素。find(doc) 返回元素或 [x, y]（应用内坐标）；
  // 元素消失后沿用最后一次看到的位置，这样点击后的涟漪不会跳走。
  const seen = new Map()
  function inApp(key, find, fx = 420, fy = 350) {
    return () => {
      let hit = null
      try { hit = frame.contentDocument && find(frame.contentDocument) } catch (_) {}
      if (Array.isArray(hit)) seen.set(key, hit)
      else if (hit) {
        const r = hit.getBoundingClientRect()
        if (r.width || r.height) seen.set(key, [r.left + r.width / 2, r.top + r.height / 2])
      }
      const p = seen.get(key) || [fx, fy]
      return [pos.x + p[0], pos.y + p[1]]
    }
  }
  const q = (sel) => (d) => d.querySelector(sel)
  const all = (d, sel) => [...d.querySelectorAll(sel)]
  const byText = (d, sel, text) => all(d, sel).find(e => e.textContent.trim() === text)
  const hasText = (d, sel, text) => all(d, sel).find(e => e.textContent.includes(text))
  // 落点：滑块当前值所在的位置
  const sliderIn = (key, find) => inApp(key, (d) => {
    const el = find(d)
    if (!el) return null
    const r = el.getBoundingClientRect()
    const k = (el.value - el.min) / (el.max - el.min)
    return [r.left + 8 + (r.width - 16) * k, r.top + r.height / 2]
  })
  // 落点：舞台自己画的元素（世界坐标）
  function onStage(sel, fx, fy) {
    return () => {
      const el = $(sel)
      if (!el) return [fx, fy]
      const r = el.getBoundingClientRect(), w = world.getBoundingClientRect()
      const k = w.width / 1920 || 1
      return [(r.left - w.left + r.width / 2) / k, (r.top - w.top + r.height / 2) / k]
    }
  }
  const pt = (x, y) => () => [x, y]

  const TAB = (name) => inApp('tab-' + name, d => byText(d, '#settings-modal .tab', name), 300, 110)
  const P = {
    rest: pt(960, 610),
    gear: inApp('gear', q('button[title="设置"]'), 640, 25),
    theme: inApp('theme', q('.theme-toggle'), 600, 25),
    save: inApp('save', q('#settings-modal .modal-footer .btn-primary'), 610, 610),
    body: inApp('body', q('#settings-modal .modal-body'), 420, 380),
    prov: inApp('prov', q('.select-shell'), 420, 330),
    provList: inApp('provList', q('.provider-dropdown-portal'), 420, 460),
    provItem: inApp('provItem', d => hasText(d, '.provider-dropdown-portal .dropdown-item', 'OpenRouter'), 420, 480),
    key: inApp('key', q('.api-tab-pane input[type=password]'), 420, 420),
    refresh: inApp('refresh', q('button[title="刷新模型列表"]'), 620, 170),
    test: inApp('test', q('button[title="测试模型连通性"]'), 650, 170),
    model: inApp('model', q('.model-tab .model-select'), 420, 210),
    modelItem: (name) => inApp('mi-' + name, d => all(d, '.model-dropdown-portal .dropdown-item').find(e => e.querySelector('.model-name')?.textContent.trim() === name), 420, 320),
    cat: (name) => inApp('cat-' + name, d => byText(d, '.cat-btn', name), 300, 330),
    domain: (name) => inApp('dom-' + name, d => all(d, '.domain-item').find(e => e.querySelector('.item-title')?.textContent.trim() === name), 420, 400),
    screen: inApp('screen', q('select.screen-select'), 420, 470),
    pill: inApp('pill', d => hasText(d, '.mode-pills .pill', '框选区域'), 300, 400),
    regionBtn: inApp('regionBtn', d => all(d, '.btn-refresh-preview').find(e => e.textContent.includes('框选')), 640, 470),
    gray: inApp('gray', d => all(d, '.setting-title').find(e => e.textContent.trim() === '灰度模式')?.closest('.setting-row')?.querySelector('.switch'), 640, 560),
    quality: sliderIn('quality', q('.compression-params input[type=range]')),
    importBtn: inApp('importBtn', q('.sk-btn.primary'), 240, 250),
    skillSwitch: (name) => inApp('sw-' + name, d => all(d, '.sk-item').find(e => e.querySelector('.sk-name')?.textContent.trim() === name)?.querySelector('.sk-switch'), 620, 330),
    skillName: (name) => inApp('sn-' + name, d => all(d, '.sk-name').find(e => e.textContent.trim() === name), 300, 380),
    upload: inApp('upload', q('.upload-card .upload-btn'), 420, 420),
    copy: inApp('copy', q('#content .code-copy-btn'), 760, 300),
    // 光标停在按钮右上角：悬停进度条在按钮底边，别让箭头挡住
    scrollUp: inApp('scrollUp', d => {
      const r = d.querySelector('[data-toolbar-action="scroll_up"]')?.getBoundingClientRect()
      return r && r.width ? [r.left + r.width * 0.86, r.top + r.height * 0.34] : null
    }, 130, 25),
    hist: (i) => inApp('hist-' + i, d => all(d, '.history-item')[i], 110, 150 + i * 90),
    histMenu: (i) => inApp('hm-' + i, d => all(d, '.history-item')[i]?.querySelector('.menu-trigger'), 190, 130 + i * 90),
    exportItem: inApp('exportItem', d => hasText(d, '.history-menu .menu-item', '导出为图片'), 280, 240),
    record: inApp('record', d => all(d, '.shortcut-item').find(e => e.querySelector('span')?.textContent.trim() === '折叠/展开面板')?.querySelector('.btn-record'), 640, 290),
    hoverSlider: sliderIn('hoverSlider', q('#hover-trigger-slider')),
    seg: (name) => inApp('seg-' + name, d => byText(d, '.iv-seg-btn', name), 300, 200),
    silence: sliderIn('silence', q('#interview-silence-slider')),
    ivModel: inApp('ivModel', q('.iv-model-row .model-select'), 420, 420),
    asr: inApp('asr', q('input.iv-input[type=password]'), 420, 520),
    pause: inApp('pause', d => hasText(d, '.iv-action-btn', '收音'), 700, 80),
    // 舞台上的落点
    through: pt(1196, 318),
    exam: pt(380, 560),
    step3: onStage('#world .xsteps span:nth-child(3)', 380, 112),
    popRow: onStage('#selpop div:nth-child(2)', 1480, 640),
    dlgRow: onStage('#fdlg .frow.pick', 1500, 480),
    dlgOpen: onStage('#fdlg .obtn.pri', 1640, 690),
    mdRow: onStage('#explorer .mdrow', 300, 330),
  }

  const PATH = [{ t: 4.4, to: P.rest }]
  const CLICKS = []
  const HOLDS = [] // 鼠标按住不放的区间（拖拽）
  function move(t, to, dur = 0.55) {
    const prev = PATH[PATH.length - 1]
    PATH.push({ t: Math.max(prev.t, t - dur), to: prev.to }, { t, to })
  }
  function click(t, to, dur) { move(t, to, dur); CLICKS.push(t) }
  function drag(a, b, to) { move(a, to); HOLDS.push([a, b]); PATH.push({ t: b, to }) }

  // 首次配置
  click(T.gear1, P.gear)
  click(T.provOpen, P.prov)
  move(T.provOpen + 0.45, P.provList, 0.3)
  click(T.provPick, P.provItem, 0.35)
  click(T.keyFocus, P.key, 0.45)
  click(T.tabModel, TAB('模型'))
  click(T.refresh, P.refresh)
  click(T.modelOpen, P.model)
  click(T.modelPick, P.modelItem(MODEL), 0.5)
  click(T.test, P.test)
  click(T.cat1, P.cat('学术考试'))
  click(T.cat2, P.cat('编程笔试'))
  click(T.domain, P.domain('Java 笔试'))
  click(T.tabShot, TAB('截图'))
  click(T.screenOpen, P.screen)
  click(T.screenPick, P.popRow, 0.45)
  click(T.modeRegion, P.pill)
  click(T.regionBtn, P.regionBtn)
  move(T.dragA, pt(SEL.x0, SEL.y0), 0.7)
  HOLDS.push([T.dragA, T.dragB + 0.05])
  move(T.dragB, pt(SEL.x1, SEL.y1), T.dragB - T.dragA)
  move(T.downA, P.body, 0.8)
  click(T.gray, P.gray, 0.4)
  drag(T.qualA, T.qualB, P.quality)
  move(T.upA, P.body, 0.25)
  click(T.save1, P.save)
  // 截图 / 解题
  move(H + 1.5, P.exam, 0.9)
  click(T.copy, P.copy, 0.8)
  // 技能 / 简历
  click(T.gear2, P.gear)
  click(T.tabSkill, TAB('技能'))
  click(T.importBtn, P.importBtn)
  click(T.pickZip, P.dlgRow)
  click(T.openZip, P.dlgOpen)
  click(T.sqlOn, P.skillSwitch('SQL 调优清单'), 0.7)
  click(T.algoOpen, P.skillName('算法题解（Java）'))
  click(T.tabResume, TAB('简历'), 0.7)
  click(T.upload, P.upload)
  click(T.pickPdf, P.dlgRow)
  click(T.openPdf, P.dlgOpen)
  click(T.save2, P.save, 0.8)
  // 隐身
  move(X + 5.5, pt(980, 640), 0.3)
  click(T.editor, P.through, 0.6)
  move(T.hoverA, P.scrollUp, 0.8)
  move(T.hoverB + 0.6, pt(1010, 600), 0.6)
  // 存储复盘
  click(T.nextQ, P.step3, 0.8)
  move(T.shotQ3 - 0.2, pt(820, 520), 0.6)
  click(T.clickMd, P.mdRow, 0.7)
  // 外观、历史与导出
  click(T.light, P.theme, 0.8)
  click(T.selA, P.hist(1), 0.8)
  click(T.menu, P.histMenu(1))
  click(T.export, P.exportItem, 0.5)
  click(T.selB, P.hist(0), 0.5)
  // 快捷键与偏好
  click(T.gear3, P.gear)
  click(T.tabGeneral, TAB('常规'))
  click(T.rec, P.record)
  move(T.genDownA, P.body, 0.5)
  drag(T.hoverDragA, T.hoverDragB, P.hoverSlider)
  click(T.save3, P.save, 0.8)
  click(T.dark, P.theme, 0.7)
  // 面试
  move(I + 0.6, pt(1010, 620), 0.1)
  click(T.mic, P.seg('麦克风'), 0.7)
  click(T.sys, P.seg('系统音频'))
  drag(T.silA, T.silB, P.silence)
  click(T.ivModelOpen, P.ivModel, 0.5)
  click(T.ivModelPick, P.modelItem(IV_MODEL), 0.5)
  click(T.asrFocus, P.asr, 0.5)
  click(T.save4, P.save, 0.6)
  move(T.f5b - 0.2, pt(1000, 640), 0.7)
  click(T.pause, P.pause, 0.8)

  function cursorAt(t) {
    if (t <= PATH[0].t) return PATH[0].to()
    for (let i = 1; i < PATH.length; i++) {
      if (t <= PATH[i].t) {
        const a = PATH[i - 1].to(), b = PATH[i].to()
        const p = eio(prog(t, PATH[i - 1].t, PATH[i].t))
        return [lerp(a[0], b[0], p), lerp(a[1], b[1], p)]
      }
    }
    return PATH[PATH.length - 1].to()
  }

  const ripples = $('#ripples')
  ripples.innerHTML = CLICKS.map(() => '<div class="ripple"></div>').join('')

  // ==================== Cue 的状态 ====================

  // 设置弹窗的四次打开：首次配置 / 技能与简历 / 外观与快捷键 / 面试
  const SESSIONS = [
    { a: T.gear1, b: T.save1, tabs: [[T.gear1, 'api'], [T.tabModel, 'model'], [T.tabShot, 'screenshot']] },
    { a: T.gear2, b: T.save2, tabs: [[T.gear2, 'screenshot'], [T.tabSkill, 'skill'], [T.tabResume, 'resume']] },
    { a: T.gear3, b: T.save3, tabs: [[T.gear3, 'resume'], [T.tabGeneral, 'general']] },
    { a: T.f5a, b: T.save4, tabs: [[T.f5a, 'interview']] },
  ]

  function settingsAt(t) {
    const ses = SESSIONS.find(x => within(t, x.a, x.b))
    if (!ses) return null
    const tab = step(t, ses.tabs)
    const scroll = tab === 'screenshot' ? eio(prog(t, T.downA, T.downB)) - eio(prog(t, T.upA, T.upB))
      : tab === 'general' ? eio(prog(t, T.genDownA, T.genDownB))
      : tab === 'interview' ? eio(prog(t, T.silB + 0.1, T.ivModelOpen - 0.2))
      : tab === 'skill' ? eio(prog(t, T.algoOpen + 0.2, T.algoOpen + 0.8))
      : 0
    return {
      tab, scroll,
      provider: within(t, T.provOpen, T.provPick),
      providerScroll: eio(prog(t, T.provOpen + 0.5, T.provPick - 0.4)),
      modelDrop: within(t, T.modelOpen, T.modelPick) ? 'main' : within(t, T.ivModelOpen, T.ivModelPick) ? 'iv' : null,
      search: typed('claude', prog(t, T.searchA, T.searchB)),
      category: step(t, [[0, '通用模式'], [T.cat1, '学术考试'], [T.cat2, '编程笔试']]),
      selecting: within(t, T.regionBtn, T.regionDone),
    }
  }

  // 在设置弹窗里改、点「保存」才生效的配置项
  function editAt(t) {
    return {
      baseURL: step(t, [[0, 'https://api.openai.com/v1'], [T.provPick, 'https://openrouter.ai/api/v1']]),
      apiKey: typed(API_KEY, prog(t, T.keyA, T.keyB)),
      model: step(t, [[0, ''], [T.loaded, MODEL_FIRST], [T.modelPick, MODEL]]),
      domainId: step(t, [[0, 'gen-solver'], [T.domain, 'dev-java-exam']]),
      screenshotMode: t >= T.modeRegion ? 'region' : 'fullscreen',
      grayscale: t < T.gray,
      compressionQuality: Math.round(lerp(80, 90, eio(prog(t, T.qualA, T.qualB)))),
      resumePath: t >= T.openPdf ? RESUME_PATH : '',
      resumeContent: t >= T.parsed ? RESUME_MD : '',
      hoverTriggerMs: t >= (T.hoverDragA + T.hoverDragB) / 2 ? 2000 : 1500,
      interviewAudioSource: within(t, T.mic, T.sys) ? 'mic' : 'system',
      interviewSilenceMs: 800 - 100 * Math.round(2 * prog(t, T.silA, T.silB)),
      interviewModel: t >= T.ivModelPick ? IV_MODEL : '',
      asrApiKey: typed(ASR_KEY, prog(t, T.asrA, T.asrB)),
    }
  }
  // 不经「保存」直接生效的：显示器、框选区域
  function directAt(t) {
    const on = t >= T.regionDone
    return {
      screenIndex: t >= T.screenPick ? 1 : 0,
      regionX: on ? SEL.x0 : 0, regionY: on ? SEL.y0 : 0,
      regionW: on ? SEL.x1 - SEL.x0 : 0, regionH: on ? SEL.y1 - SEL.y0 : 0,
    }
  }
  const lastSave = (t) => Math.max(-1, ...SAVES.filter(x => x <= t))

  // 截图预览此刻该显示的画面（后端应答）
  function previewAt(t) {
    if (t >= T.qualB) return { img: SHOTS[0], size: '171 KB · 678×648' }
    if (t >= T.gray) return { img: SHOTS[0], size: '142 KB · 678×648' }
    if (t >= T.regionDone) return { img: IMG + 'q2-a-gray.jpg', size: '96 KB · 678×648' }
    if (t >= T.screenPick) return { img: IMG + 'screen2-gray.jpg', size: '238 KB · 1920×1080' }
    return { img: IMG + 'screen1-gray.jpg', size: '412 KB · 2560×1440' }
  }

  function shotsAt(t) {
    if (t >= SB.send) return []
    if (t >= T.shotQ3 + FLY) return [SHOT3]
    if (t >= SA.send) return []
    if (t >= T.del) return SHOTS.slice(0, 2)
    return SHOTS.slice(0, count(t, [T.shot1 + FLY, T.shot2 + FLY, T.shot3 + FLY]))
  }
  const CAPS = [[T.shot1, SHOTS[0]], [T.shot2, SHOTS[1]], [T.shot3, SHOTS[2]], [T.shotQ3, SHOT3]]

  function solveAt(t) {
    if (t < SA.send) return null
    const item = (x, thinking, answer, shot) => ({
      time: x.time, shot, done: t >= x.done,
      thinking: x.think != null ? typed(thinking, prog(t, x.think, x.answer - 0.1)) : '',
      answer: typed(answer, prog(t, x.answer, x.done)),
    })
    const items = []
    if (t >= SB.answer) items.push(item(SB, '', ANSWER3, SHOT3))
    if (t >= SA.think) items.push(item(SA, THINKING, ANSWER, SHOTS[0]))
    const cur = t >= SB.send ? SB : SA
    const active = within(t, T.selA, T.selB) ? 1 : 0
    let scroll = 1
    if (t < SB.answer) {
      // 第 2 题：Alt+PgUp / PgDn 翻页，之后悬停在「上翻」按钮上连续滚动
      const pages = T.pgUp.reduce((n, a) => n + eo(prog(t, a, a + 0.28)), 0) - T.pgDn.reduce((n, a) => n + eo(prog(t, a, a + 0.28)), 0)
      scroll = clamp(1 - 0.15 * pages - 0.62 * prog(t, T.hoverFire, T.hoverB))
    } else if (active === 1) scroll = 0
    return {
      items, active, scroll,
      loading: within(t, cur.send, cur.think ?? cur.answer),
      thinking: cur.think != null && within(t, cur.think, cur.answer),
      solving: t < cur.done,
      menu: within(t, T.menu, T.export) ? 1 : null,
      copied: within(t, T.copy, T.copy + 1.8),
    }
  }

  function interviewAt(t) {
    const on = t >= T.f5b
    const iv = { on, listening: on && t < T.pause, answering: false, transcript: '', rounds: [] }
    for (const r of ROUNDS) {
      if (within(t, r.speak[0] + 0.1, r.ask)) iv.transcript = typed(r.q, prog(t, r.speak[0] + 0.1, r.speak[1]))
      if (t < r.ask) continue
      iv.rounds.push({ question: r.q, skills: t >= r.ask + 0.2 ? r.skills : [], answer: typed(r.a, prog(t, r.ans[0], r.ans[1])), done: t >= r.ans[1] })
      if (within(t, r.ask + 0.25, r.ans[1])) iv.answering = true
    }
    return iv
  }

  // 片中出现的 toast：[时刻, 文案, 类型, 停留秒数]
  const TOASTS = [
    [T.loaded, '已加载 14 个模型', 'success', 2],
    [T.tested, '连接测试成功', 'success', 2],
    [T.save1, '设置已保存', 'success', 2],
    [SA.send + 0.6, '已加载技能：算法题解（Java）', 'info', 3],
    [T.openZip, '已导入技能：Redis 速查', 'success', 2],
    [T.openPdf, '已选择文件，正在自动解析...', 'info', 2],
    [T.parsed, '简历解析成功', 'success', 2],
    [T.save2, '设置已保存', 'success', 2],
    [T.f6, '快捷键屏蔽已开启：软件快捷键及其组合键对外不可见（F6 恢复）', 'warning', 3],
    [T.f4, '存储复盘已开启：截图与解答将存入文档/Cue/复盘（F4 关闭）', 'success', 3],
    [SB.send + 0.4, '已加载技能：后端八股速答', 'info', 3],
    [T.recorded, '快捷键已保存', 'success', 2],
    [T.save3, '设置已保存', 'success', 2],
    [T.f5a, '请先配置语音识别 API Key', 'warning', 2],
    [T.save4, '设置已保存', 'success', 2],
  ]

  function appState(t) {
    const set = settingsAt(t)
    const save = lastSave(t)
    const iv = interviewAt(t)
    const solve = solveAt(t)
    const view = t >= T.f5b ? 'interview' : solve ? 'solve' : 'welcome'
    const transparency = +(0.05 * (count(t, T.opUp) - count(t, T.opDown))).toFixed(2)
    const steps = (a) => Math.floor(prog(t, a, a + 0.8) * 20)
    const s = {
      present: t >= C.bar,
      drop: eo(prog(t, C.bar, C.bar + 0.6)),
      collapsed: t < T.f7 + 0.1,
      grow: eio(prog(t, T.f7 + 0.1, T.f7 + 0.55)),
      pos: { x: APP.x - 10 * steps(T.moveL) + 10 * steps(T.moveR), y: APP.y },
      theme: within(t, T.light, T.dark) ? 'light' : 'dark',
      hover: within(t, T.hoverA, T.hoverB) ? 'scroll_up' : null,
      hoverProgress: t < T.hoverFire ? prog(t, T.hoverA, T.hoverFire) : 0,
      shots: shotsAt(t),
      view, solve,
      settings: set,
      conf: { temp: { ...editAt(set ? t : save), ...directAt(t) }, saved: { ...editAt(save), ...directAt(t) } },
      preview: previewAt(t),
      models: t >= T.loaded,
      loadingModels: within(t, T.refresh, T.loaded),
      testing: within(t, T.test, T.tested),
      connection: within(t, T.tested, T.save1) ? { type: 'success', icon: '✓', message: `模型 ${MODEL} 连接成功` } : null,
      parsing: within(t, T.openPdf, T.parsed),
      recording: within(t, T.rec, T.recorded),
      toolbarKey: t >= T.recorded ? 'Alt+Q' : 'F7',
      skill: { imported: t >= T.openZip, importing: within(t, T.importBtn, T.openZip), on: { 'sql-tuning': t >= T.sqlOn }, open: t >= T.algoOpen ? 'algo-java' : '' },
      transparency,
      clickThrough: within(t, T.f10on, T.f10off),
      shield: t >= T.f6,
      review: t >= T.f4,
      toasts: [],
      interview: iv,
      status: t < T.save1 ? '未配置' : iv.on ? (iv.listening ? '聆听中' : '已暂停') : solve?.solving ? '正在思考...' : solve ? '解题完成' : '已连接',
      // 桌面
      exam: { q: t >= T.nextQ ? 3 : 2, scroll: 0.5 * eio(prog(t, H + 1.7, H + 2.4)) + 0.5 * eio(prog(t, H + 3.7, H + 4.4)) - eio(prog(t, H + 6.6, H + 7.3)) },
      meet: t >= I,
      speaking: Math.max(...ROUNDS.map(r => env(t, r.speak[0], r.speak[1], 0.2, 0.3))),
      typedLine: typed(TYPED_LINE, prog(t, T.typeA, T.typeB)),
      caret: t >= T.editor,
    }

    const toast = (a, text, type, hold) => {
      if (within(t, a, a + hold + 0.4)) s.toasts.push({ id: a, text, type, show: within(t, a + 0.05, a + hold) })
    }
    for (const x of TOASTS) toast(...x)
    // 连按透明度快捷键：合并成一条随数值更新的提示
    for (const presses of [T.opUp, T.opDown]) {
      if (within(t, presses[0], presses[8] + 2)) toast(presses[0], `窗口透明度 ${Math.round(transparency * 100)}%`, 'info', presses[8] - presses[0] + 1.6)
    }
    return s
  }

  // 截图态的补丁（README 用图）：在时间线某一刻的基础上改几个字段
  let override = null

  // ---- 同步进真实应用 ----

  const put = (obj, key, val) => { if (obj[key] !== val) obj[key] = val }
  let applied = {}
  const once = (key, val, fn) => { const k = JSON.stringify(val); if (applied[key] !== k) { applied[key] = k; fn() } }

  function syncApp(s) {
    const ui = cue.store('ui'), st = cue.store('settings'), sol = cue.store('solution'), iv = cue.store('interview')
    if (!ui || !st || !sol || !iv) return

    put(ui, 'isCollapsed', s.collapsed)
    put(ui, 'hoverToolbarAction', s.hover)
    put(ui, 'hasStarted', s.view !== 'welcome')
    put(ui, 'showInterview', s.view === 'interview')
    put(ui, 'isStealthMode', true)
    put(ui, 'isClickThrough', s.clickThrough)
    put(ui, 'isShieldOn', s.shield)
    put(ui, 'isReviewOn', s.review)

    // 设置弹窗：先开关，再覆盖成这一刻的配置
    if (!!s.settings !== ui.showSettings) {
      if (s.settings) st.openSettings()
      else { st.isResumeParsing = false; st.closeSettings(); ui.showSettings = false }
    }
    if (s.settings) put(ui, 'activeTab', s.settings.tab)
    for (const k in s.conf.saved) put(st.settings, k, s.conf.saved[k])
    for (const k in s.conf.temp) put(st.tempSettings, k, s.conf.temp[k])
    put(st, 'resumeRawContent', s.conf.temp.resumeContent)
    put(st, 'isResumeParsing', s.parsing)
    put(st, 'recordingAction', s.recording ? 'toggle_toolbar' : null)
    if (s.recording) put(st, 'recordingText', '请按键...')
    once('toolbarKey', s.toolbarKey, () => {
      const key = { comboID: s.toolbarKey === 'F7' ? '118' : '81+164', keyName: s.toolbarKey }
      st.shortcuts.toggle_toolbar = { ...key }
      st.tempShortcuts.toggle_toolbar = { ...key }
    })
    if (st.tempShortcuts.toggle_toolbar?.keyName !== s.toolbarKey) applied.toolbarKey = ''
    put(st, 'statusText', s.status)
    put(st, 'statusIcon', s.status === '未配置' ? '!' : '✓')

    once('models', s.models, () => { ui.availableModels = s.models ? [...cue.models] : [] })
    if (ui.availableModels.length !== (s.models ? cue.models.length : 0)) applied.models = ''
    put(ui, 'isLoadingModels', s.loadingModels)
    put(ui, 'isTestingConnection', s.testing)
    if (JSON.stringify(ui.connectionStatus) !== JSON.stringify(s.connection)) ui.connectionStatus = s.connection && { ...s.connection }

    cue.script.preview = s.preview
    const ids = cue.skills.map(x => x.id)
    cue.script.skills = s.skill.imported ? ids : ids.filter(id => id !== 'redis-cheatsheet')

    // 截图坞：按张数增减，和真实后端一样逐张发事件
    const have = applied.shots || []
    if (have.join() !== s.shots.join()) {
      if (!s.shots.length) cue.emit('screenshots-cleared')
      else if (s.shots.slice(0, have.length).join() === have.join()) {
        for (let i = have.length; i < s.shots.length; i++) cue.emit('screenshot-taken', s.shots[i], i + 1)
      } else if (have.slice(0, s.shots.length).join() === s.shots.join()) {
        for (let i = have.length - 1; i >= s.shots.length; i--) cue.emit('screenshot-removed', i, i)
      } else {
        cue.emit('screenshots-cleared')
        s.shots.forEach((img, i) => cue.emit('screenshot-taken', img, i + 1))
      }
      applied.shots = [...s.shots]
    }

    once('solve', s.solve && { ...s.solve, scroll: 0, menu: 0, copied: 0 }, () => {
      const v = s.solve
      const items = v ? v.items : []
      sol.history = items.map(it => ({ time: it.time, rounds: [{ userScreenshot: it.shot, thinking: it.thinking, thinkingStatus: 'Thinking Process', aiResponse: it.answer, error: null }] }))
      const head = items[0]
      sol.streamingHtml = head && !head.done && head.answer && v.active === 0 ? sol.renderMarkdown(head.answer) : ''
      sol.activeHistoryIndex = v ? Math.min(v.active, Math.max(0, items.length - 1)) : 0
      sol.isLoading = !!v?.loading
      sol.isSolving = !!v?.solving
      sol.isThinking = !!v?.thinking
    })

    once('toasts', s.toasts, () => { ui.toasts = s.toasts.map(x => ({ ...x })) })

    if (applied.transparency !== s.transparency) {
      applied.transparency = s.transparency
      st.settings.transparency = s.transparency
      st.tempSettings.transparency = s.transparency
      st.applyTransparency(1 - s.transparency)
    }

    once('interview', s.interview, () => {
      const v = s.interview
      iv.isOn = v.on
      iv.isListening = v.listening
      iv.isAnswering = v.answering
      iv.transcript = v.transcript
      iv.rounds = v.rounds.map(r => ({ question: r.question, skills: r.skills, answer: r.done ? r.answer : '', html: r.answer ? sol.renderMarkdown(r.answer) : '' }))
      ui.isInterviewOn = v.on
      ui.isInterviewListening = v.listening
    })
  }

  // 没有对应 store 字段的界面状态：去点真实的按钮，把界面对齐到 s。
  // 每次调用最多点一下——点完要等 Vue 刷新，下一帧再接着对。
  let clickGuard = 0
  function settleApp(s) {
    const doc = frame.contentDocument
    if (!doc || !cue) return
    clickGuard = Math.max(0, clickGuard - 1)
    const hit = (el) => { if (!el || clickGuard) return false; el.click(); clickGuard = 2; return true }
    const converge = () => {
      // 主题
      if (s.theme && doc.documentElement.getAttribute('data-theme') !== s.theme && hit(doc.querySelector('.theme-toggle'))) return

      const set = s.settings
      if (set) {
        // 提供商下拉。开合看 .open——收起过渡期间下拉节点还在，不能拿它当依据
        const shell = doc.querySelector('.select-shell')
        if (shell && shell.classList.contains('open') !== set.provider && hit(shell)) return
        const prov = doc.querySelector('.provider-dropdown-portal')
        if (prov) prov.scrollTop = (prov.scrollHeight - prov.clientHeight) * set.providerScroll

        // 模型下拉（主模型 / 面试模型共用一个组件）
        const open = doc.querySelector('.model-select.open')
        const which = open ? (open.closest('.iv-model-row') ? 'iv' : 'main') : null
        if (which !== set.modelDrop) {
          if (open) {
            const want = which === 'iv' ? s.conf.temp.interviewModel : s.conf.temp.model
            const item = all(doc, '.model-dropdown-portal .dropdown-item').find(e => e.querySelector('.model-name')?.textContent.trim() === want)
            if (hit(item || open)) return
          } else if (hit(doc.querySelector(set.modelDrop === 'iv' ? '.iv-model-row .model-select' : '.model-tab .model-select'))) return
        }
        const search = doc.querySelector('.model-search-input')
        if (search && set.modelDrop === 'main' && search.value !== set.search) {
          search.value = set.search
          search.dispatchEvent(new Event('input', { bubbles: true }))
        }

        // 场景分类
        if (set.tab === 'model') {
          const cur = doc.querySelector('.cat-btn.active')
          if (cur && cur.textContent.trim() !== set.category && hit(byText(doc, '.cat-btn', set.category))) return
        }

        if (set.tab === 'screenshot') {
          // 框选区域：点真实按钮进入「框选中...」，结束时由后端事件带回区域
          const btn = all(doc, '.btn-refresh-preview').find(e => e.textContent.includes('框选'))
          const busy = btn?.textContent.includes('框选中')
          if (btn && set.selecting && !busy && hit(btn)) return
          if (btn && !set.selecting && busy) {
            cue.emit('region-selected', { x: SEL.x0, y: SEL.y0, w: SEL.x1 - SEL.x0, h: SEL.y1 - SEL.y0 })
            return
          }
          // 预览与这一刻该有的画面不一致就刷新
          const box = doc.querySelector('.preview-box')
          if (box && !box.querySelector('.preview-spinner')) {
            const img = box.querySelector('.preview-img')?.getAttribute('src') || ''
            const size = box.querySelector('.size-tag')?.textContent.trim() || ''
            if ((img !== s.preview.img || size !== s.preview.size) && hit(doc.querySelector('.btn-refresh-preview'))) return
          }
        }

        if (set.tab === 'skill') {
          // 导入：点「导入文件」后原生文件框挂起，片中「点了打开」时兑现
          if (s.skill.importing && !cue.pending.importSkill && hit(doc.querySelector('.sk-btn.primary'))) return
          if (!s.skill.importing && cue.pending.importSkill) {
            const sk = cue.skills.find(x => x.id === 'redis-cheatsheet')
            cue.pending.importSkill(s.skill.imported ? { id: sk.id, name: sk.name } : null)
            cue.pending.importSkill = null
            clickGuard = 2
            return
          }
          for (const li of all(doc, '.sk-item')) {
            const id = cue.skills.find(x => x.name === li.querySelector('.sk-name')?.textContent.trim())?.id
            const input = li.querySelector('input[type=checkbox]')
            const want = s.skill.on[id]
            if (want !== undefined && input.checked !== want && hit(input)) return
            if (!!li.querySelector('.sk-preview') !== (s.skill.open === id) && hit(li.querySelector('.sk-main'))) return
          }
        }
      }

      if (s.view === 'solve' && s.solve) {
        // 思考完成后思维链自动折叠；跳帧取景时补上这一下
        if (hit(doc.querySelector('#content .thinking-block:not(.active):not(.collapsed) .thinking-header'))) return
        const menu = doc.querySelector('.history-menu')
        if (s.solve.menu != null && !menu && hit(all(doc, '.history-item')[s.solve.menu]?.querySelector('.menu-trigger'))) return
        if (s.solve.menu == null && menu) { doc.body.click(); clickGuard = 2; return }
        const copy = doc.querySelector('#content .code-copy-btn')
        if (copy && s.solve.copied && !copy.classList.contains('copied') && hit(copy)) return
      }
    }
    converge()

    // 滚动位置、悬停进度直接写
    const content = doc.getElementById('content')
    if (content && s.solve) content.scrollTop = (content.scrollHeight - content.clientHeight) * s.solve.scroll
    const pane = s.settings && all(doc, '#settings-modal .tab-pane').find(e => e.clientHeight > 0) // 每个标签页各自滚动
    if (pane) {
      // 技能页滚到展开的那一项为止，其余标签页按比例滚到底
      const open = s.settings.tab === 'skill' && pane.querySelector('.sk-preview')?.closest('.sk-item')
      const max = pane.scrollHeight - pane.clientHeight
      const to = open ? Math.min(max, pane.scrollTop + open.getBoundingClientRect().top - pane.getBoundingClientRect().top - 10) : max
      pane.scrollTop = to * clamp(s.settings.scroll)
    }
    for (const bar of doc.querySelectorAll('.tb-progress')) {
      const on = bar.closest('[data-toolbar-action]')?.dataset.toolbarAction === s.hover
      bar.style.setProperty('--hover-progress', on ? s.hoverProgress : 0)
    }
  }

  // ==================== 一帧 ====================

  const lower = $('#lower')
  let lowerShown = null
  let hide = {}

  function renderLower(t) {
    const seg = LOWER.find(x => within(t, x.a, x.b))
    if (!seg || hide.lower) { lower.style.opacity = 0; return }
    if (lowerShown !== seg) {
      lowerShown = seg
      const m = /^(\d+)\s*·\s*(.+)$/.exec(seg.tag)
      lower.classList.toggle('nonum', !m)
      $('.lt-num', lower).textContent = m ? m[1] : ''
      $('.lt-tag', lower).textContent = m ? m[2] : seg.tag
      $('.lt-title', lower).textContent = seg.title
      $('.lt-sub', lower).textContent = seg.sub
      $('.lt-keys', lower).innerHTML = seg.keys.map(k =>
        `<div class="kg">${k.combo.map(c => `<kbd>${c}</kbd>`).join('<span class="plus">+</span>')}<em>${k.label}</em></div>`).join('')
      $('.lt-keys', lower).style.display = seg.keys.length ? '' : 'none'
    }
    const p = env(t, seg.a, seg.b)
    lower.style.opacity = p
    lower.style.transform = `translateX(${(1 - p) * -64}px)`
    ;[...lower.querySelectorAll('.kg')].forEach((node, i) => {
      const times = [].concat(seg.keys[i].at ?? [])
      node.classList.toggle('down', times.some(a => within(t, a, a + 0.11)))
    })
  }

  // 应用内元素在世界坐标中的矩形
  function appRect(sel) {
    const el = frame.contentDocument?.querySelector(sel)
    if (!el) return null
    const r = el.getBoundingClientRect()
    return { x: pos.x + r.left, y: pos.y + r.top, w: r.width, h: r.height }
  }
  const popIn = (el, p, style = {}) => Object.assign(el.style, { opacity: p, visibility: p > 0.001 ? 'visible' : 'hidden', transform: `scale(${0.96 + 0.04 * p})`, ...style })

  function renderWindows(t) {
    const off = hide.still && !hide.windows ? 0 : 1
    // 文件选择框：导入技能 / 选择简历
    const dlg = within(t, T.importBtn, T.openZip + 0.15) ? ['skill', T.importBtn, T.pickZip, T.openZip]
      : within(t, T.upload, T.openPdf + 0.15) ? ['resume', T.upload, T.pickPdf, T.openPdf] : null
    if (dlg) {
      const [kind, a, pick, b] = dlg
      fillDialog(kind)
      const d = DIALOGS[kind]
      const picked = t >= pick
      $('.frow.pick', fdlg).classList.toggle('sel', picked)
      $('[data-name]', fdlg).textContent = picked ? d.rows[d.pick][1] : ''
      popIn(fdlg, off * env(t, a + 0.12, b + 0.12, 0.18, 0.12), { left: pos.x + 100 + 'px', top: pos.y + 160 + 'px' })
    } else popIn(fdlg, 0)

    // 「截取屏幕」原生下拉
    const pop = $('#selpop')
    const sel = within(t, T.screenOpen, T.screenPick + 0.1) && appRect('select.screen-select')
    if (sel) {
      const hot = t >= T.screenPick - 0.2 ? 1 : 0
      ;[...pop.children].forEach((n, i) => n.classList.toggle('hot', i === hot))
      Object.assign(pop.style, { left: sel.x + 'px', top: sel.y + sel.h + 2 + 'px', width: sel.w + 'px',
        opacity: off * env(t, T.screenOpen + 0.05, T.screenPick + 0.1, 0.1, 0.08), visibility: 'visible' })
    } else Object.assign(pop.style, { opacity: 0, visibility: 'hidden' })

    // 复盘文件夹
    const ex = $('#explorer')
    popIn(ex, off * env(t, T.folderIn, T.folderOut, 0.3, 0.25))
    const md = t >= T.clickMd
    $('.mdrow', ex).classList.toggle('sel', md)
    const pv = eo(prog(t, T.clickMd + 0.05, T.clickMd + 0.4))
    $('.pv-empty', ex).style.opacity = md ? 0 : 1
    Object.assign($('.pv-md', ex).style, { opacity: pv, transform: `translateY(${(1 - pv) * 10}px)` })

    // 导出的图片
    const vw = $('#viewer')
    popIn(vw, off * env(t, T.viewerIn, T.viewerOut, 0.3, 0.25))
    $('[data-vimg]', vw).style.backgroundPositionY = 100 * eio(prog(t, T.viewerIn + 1.0, T.viewerOut - 0.6)) + '%'
  }

  function render(t) {
    const s = appState(t)
    if (override) override(s)
    pos = s.pos

    // --- 镜头 ---
    const cam = hide.cam || track(t, CAMERA)
    const split = hide.cam ? 0 : Math.max(0, Math.min(eio(prog(t, X, X + 0.9)), 1 - eio(prog(t, X + 4.6, X + 5.5))))
    const radius = 0
    world.style.transform = `translate(${cam[0]}px, ${cam[1]}px) scale(${cam[2]})`
    world.style.borderRadius = radius + 'px'
    world.style.boxShadow = 'none'
    const desktop = within(t, 3.6, T.outroA + 0.7) || !!hide.still
    world.style.visibility = desktop ? 'visible' : 'hidden'

    // --- 分屏对比：右边是「对方看到的画面」，同一张桌面，没有 Cue ---
    share.style.visibility = split > 0.001 ? 'visible' : 'hidden'
    share.style.transform = `translate(${lerp(1960, 974, split)}px, 205px) scale(0.47)`
    share.style.borderRadius = '0'
    share.style.boxShadow = 'none'
    const labels = env(t, X + 0.7, X + 4.6, 0.4, 0.3) * (hide.cam ? 0 : 1)
    Object.assign($('#lbl-you').style, { left: '44px', top: '138px', opacity: labels })
    Object.assign($('#lbl-share').style, { left: '974px', top: '138px', opacity: labels })
    Object.assign($('#ghost').style, { left: pos.x + 'px', top: pos.y + 'px', width: APP.w + 'px', height: APP.h + 'px',
      opacity: env(t, X + 1.5, X + 4.6, 0.4, 0.3) })

    // --- 桌面 ---
    const deskState = { meet: s.meet, speaking: s.speaking, typed: s.typedLine, caret: s.caret, q: s.exam.q, scroll: s.exam.scroll }
    for (const d of desks) updateDesk(d, t, deskState)

    // --- Cue 窗口 ---
    const size = s.collapsed ? BAR : APP
    frame.style.width = size.w + 'px'
    frame.style.height = size.h + 'px'
    const cw = s.collapsed ? BAR.w : lerp(BAR.w, APP.w, s.grow)
    const ch = s.collapsed ? BAR.h : lerp(BAR.h, APP.h, s.grow)
    Object.assign(wrap.style, {
      left: pos.x + 'px', top: pos.y + 'px', width: size.w + 'px', height: size.h + 'px',
      opacity: s.present ? s.drop : 0,
      transform: `translateY(${(1 - s.drop) * -40}px)`,
      clipPath: !s.collapsed && s.grow < 1 ? `inset(0 ${size.w - cw}px ${size.h - ch}px 0 round 14px)` : 'none',
    })
    if (cue) {
      syncApp(s)
      frame.style.colorScheme = frame.contentDocument.documentElement.getAttribute('data-theme') || 'light'
    }

    // --- 框选区域：系统级遮罩盖在一切之上，包括 Cue 的设置弹窗 ---
    const dimP = hide.still ? 0 : env(t, T.regionBtn + 0.1, T.regionDone, 0.3, 0.12)
    const dragP = eio(prog(t, T.dragA, T.dragB))
    const sel = $('#sel')
    const sw = (SEL.x1 - SEL.x0) * dragP, sh = (SEL.y1 - SEL.y0) * dragP
    $('#dim').style.opacity = dimP
    $('#dim').style.background = t < T.dragA ? 'rgba(11,18,32,.5)' : 'none'
    Object.assign(sel.style, { left: SEL.x0 + 'px', top: SEL.y0 + 'px', width: sw + 'px', height: sh + 'px', opacity: t >= T.dragA ? 1 : 0 })
    $('span', sel).textContent = `${Math.round(sw)} × ${Math.round(sh)}`

    // --- 截图：区域闪白，缩略图飞进 Cue 底部的截图坞 ---
    const cap = hide.still ? null : CAPS.find(([a]) => within(t, a, a + FLY + 0.15))
    const flash = $('#flash'), thumb = $('#thumb')
    if (cap) {
      const [a, img] = cap
      Object.assign(flash.style, { left: SEL.x0 + 'px', top: SEL.y0 + 'px', width: SEL.x1 - SEL.x0 + 'px', height: SEL.y1 - SEL.y0 + 'px',
        opacity: 0.75 * (1 - prog(t, a, a + 0.35)) })
      const fly = eio(prog(t, a + 0.05, a + FLY))
      const [dx, dy] = DOCK()
      Object.assign(thumb.style, {
        left: lerp(SEL.x0, dx - 27, fly) + 'px', top: lerp(SEL.y0, dy - 18, fly) + 'px',
        width: lerp(SEL.x1 - SEL.x0, 54, fly) + 'px', height: lerp(SEL.y1 - SEL.y0, 37, fly) + 'px',
        backgroundImage: `url(${img})`,
        opacity: t >= a + 0.05 ? 1 - prog(t, a + FLY - 0.1, a + FLY + 0.04) : 0,
      })
    } else {
      flash.style.opacity = 0
      thumb.style.opacity = 0
    }

    renderWindows(t)

    // --- 光标与点击 ---
    const cur = $('#cursor')
    const [cx, cy] = cursorAt(t)
    const curOn = hide.cursor ? 0 : hide.still ? 1
      : env(t, 4.5, T.gridA - 0.1, 0.3, 0.2) * (1 - env(t, X + 0.1, X + 5.5, 0.2, 0.2)) * (1 - env(t, I - 0.4, I + 0.5, 0.2, 0.2))
    Object.assign(cur.style, { left: cx + 'px', top: cy + 'px', opacity: curOn })
    const press = CLICKS.some(c => within(t, c - 0.05, c + 0.12)) || HOLDS.some(([a, b]) => within(t, a, b))
    $('svg', cur).style.transform = `scale(${press ? 0.86 : 1})`
    CLICKS.forEach((c, i) => {
      const node = ripples.children[i]
      const p = prog(t, c, c + 0.5)
      if (p <= 0 || p >= 1 || hide.cursor) { node.style.opacity = 0; return }
      const size = 12 + eo(p) * 62
      const [rx, ry] = cursorAt(c)
      Object.assign(node.style, { left: rx - size / 2 + 'px', top: ry - size / 2 + 'px', width: size + 'px', height: size + 'px', opacity: 1 - p })
    })

    // --- 叠加层 ---
    const zoomed = clamp((cam[2] - 1.12) / 0.2)
    $('#vignette').style.opacity = hide.lower ? 0 : zoomed * (1 - split) * (1 - prog(t, T.gridA, T.gridA + 0.4))
    $('#dip').style.opacity = hide.still ? 0 : Math.max(0, 1 - Math.abs(t - I) / 0.32)
    renderLower(t)
    renderMore(hide.still ? -1 : t)
    renderIntro(hide.still ? 99 : t)
    renderOutro(hide.still ? -1 : t)
    return s
  }
  const DOCK = inApp('dock', d => d.querySelector('.screenshot-dock .dock-thumbs') || d.querySelector('.screenshot-dock'), 420, 633)

  // ==================== 启动 ====================

  function fit() {
    if (CAPTURE) return
    const k = Math.min(innerWidth / 1920, (innerHeight - 56) / 1080)
    $('#stage').style.transform = `scale(${k})`
  }
  addEventListener('resize', fit)
  fit()

  function loadApp(theme) {
    cue = null
    applied = {}
    seen.clear()
    return new Promise((resolve) => {
      frame.onload = async () => {
        const w = frame.contentWindow
        for (let i = 0; i < 400; i++) {
          if (w.__cue?.store('ui') && w.document.querySelector('.top-bar')) break
          await new Promise(r => setTimeout(r, 25))
        }
        // 片中的 toast 全部由时间轴给出，应用自己弹的一律不要
        w.__cue.store('ui').showToast = () => {}
        await w.__cue.tick(WARMUP)
        await new Promise(r => setTimeout(r, 120))
        cue = w.__cue
        for (const src of [...SHOTS, SHOT3]) new w.Image().src = src
        resolve()
      }
      frame.src = `/?theme=${theme}${CAPTURE ? '&vclock=1' : ''}`
    })
  }

  const realFrame = () => new Promise(r => requestAnimationFrame(() => r()))

  // 逐帧渲染入口（render.py 调用）：把应用时钟推到 t，等界面落定后冻结动画
  // clock 是应用时钟（秒），缺省与片内时间一致；静帧取景时两者分开走
  async function frameAt(t, clock = t) {
    const s = render(t)
    await cue.tick(WARMUP + clock * 1000)
    settleApp(s)
    await cue.tick(WARMUP + clock * 1000)
    await realFrame()
    render(t) // 光标落点依赖应用内的元素位置，界面更新后再对一次
    cue.freeze()
    await realFrame()
  }

  // README 用图：时间线上某一刻 + 补丁。opts 见 render.py 的 STILLS
  //   mark / t: 取景时刻（MARKS 里的名字，或直接给秒数）
  //   cam: null 沿用时间线镜头（含分屏对比），缺省为全景
  //   bare: 去掉桌面，换成纯净的渐变底（配合裁切使用）
  //   noapp: 不显示 Cue（给桌面本身拍照）
  //   windows: 保留舞台画的系统窗口（资源管理器等），缺省隐藏
  // 返回 Cue 窗口与设置弹窗在画面中的位置，供裁切
  const BACKDROP = {
    dark: '#0f1726',
    light: '#d9e3f0',
  }
  const BANNER = {
    zh: {
      tag: '截图即问<b>·</b>听题即答<b>·</b>悬浮隐身',
      sub: '开源桌面 AI 助手：笔试截图，流式给出思路与代码；面试实时听题作答。窗口置顶、可穿透，对屏幕共享不可见。',
      chips: ['截图解题', '面试模式', '技能 Skills', '隐身 · 穿透'],
      foot: 'Windows / macOS · Wails + Go + Vue 3 · 基于 Q-Solver 二次开发',
    },
    en: {
      tag: 'See it. Hear it. <b>Answer it.</b>',
      sub: 'An open-source desktop AI copilot. Screenshot a problem for a streamed solution, or let it listen and answer live — in a floating window that screen sharing can\'t see.',
      chips: ['Screenshot solving', 'Interview mode', 'Skills', 'Stealth'],
      foot: 'Windows / macOS · Wails + Go + Vue 3 · A fork of Q-Solver',
    },
  }
  const bannerHTML = (lang) => {
    const b = BANNER[lang] || BANNER.zh
    return `<div class="bn ${lang}">
      <div class="brand">${logoSVG('b')}<div class="word">&nbsp;Cue</div></div>
      <div class="bn-tag">${b.tag}</div>
      <div class="bn-sub">${b.sub}</div>
      <div class="chips">${b.chips.map(c => `<div class="chip sq">${c}</div>`).join('')}</div>
      <div class="bn-foot">${b.foot}</div>
    </div>`
  }
  let stillClock = DUR + 10
  async function still(opts) {
    if ((opts.theme || 'dark') !== theme) { theme = opts.theme || 'dark'; await loadApp(theme) }
    const t = opts.mark ? MARKS[opts.mark] : opts.t
    hide = { lower: true, cursor: !opts.cursor, cam: opts.cam === null ? null : opts.cam || FULL, still: true, windows: !!opts.windows }
    override = (s) => {
      Object.assign(s, opts.state || {})
      if (opts.solve && s.solve) Object.assign(s.solve, opts.solve)
      if (opts.settings && s.settings) Object.assign(s.settings, opts.settings)
      s.present = !opts.noapp; s.drop = 1; s.grow = 1; s.theme = theme
    }
    desks[0].style.visibility = opts.bare ? 'hidden' : ''
    const banner = $('#banner')
    banner.style.display = opts.banner ? '' : 'none'
    if (opts.banner) banner.innerHTML = bannerHTML(opts.banner)
    world.style.background = opts.bare ? BACKDROP[theme] : ''
    // 片内时间定格在 t，应用时钟继续走：让弹窗过渡、异步加载落定
    for (let i = 0; i < 48; i++) await frameAt(t, stillClock += 1 / FPS)
    const r = wrap.getBoundingClientRect()
    const k = r.width / wrap.offsetWidth
    const rect = (el) => {
      if (!el) return null
      const b = el.getBoundingClientRect()
      return { x: r.left + b.left * k, y: r.top + b.top * k, w: b.width * k, h: b.height * k }
    }
    return { app: { x: r.left, y: r.top, w: r.width, h: r.height },
      modal: rect(frame.contentDocument.querySelector('#settings-modal .modal-content')) }
  }

  // 时间轴导出：music.py 据此对齐音效与段落，render.py 据此分段与取静帧
  const keyTimes = LOWER.flatMap(x => x.keys.flatMap(k => [].concat(k.at ?? []))).sort((a, b) => a - b)
  const span = (k, a, b) => ({ k, t: a, d: +(b - a).toFixed(3) })
  const CUES = [
    { k: 'drop', t: C.bar }, { k: 'expand', t: T.f7 + 0.1 },
    ...SESSIONS.flatMap(x => [{ k: 'open', t: x.a }, { k: 'close', t: x.b }]),
    ...TOASTS.map(([t, , type]) => ({ k: type === 'warning' ? 'warn' : 'toast', t })),
    span('type', T.keyA, T.keyB), span('type', T.searchA, T.searchB), span('type', T.asrA, T.asrB),
    span('typeDesk', T.typeA, T.typeB),
    span('spin', T.refresh, T.loaded), span('spin', T.test, T.tested), span('spin', T.openPdf, T.parsed),
    { k: 'whoosh', t: T.regionBtn }, span('drag', T.dragA, T.dragB), { k: 'whoosh', t: T.regionDone + 0.15 },
    ...CAPS.map(([t]) => ({ k: 'shutter', t })),
    { k: 'remove', t: T.del },
    span('think', SA.think, SA.answer), span('stream', SA.answer, SA.done), { k: 'done', t: SA.done },
    span('stream', SB.answer, SB.done), { k: 'done', t: SB.done },
    { k: 'send', t: SA.send }, { k: 'send', t: SB.send },
    { k: 'pop', t: T.importBtn + 0.1 }, { k: 'pop', t: T.upload + 0.1 }, { k: 'pop', t: T.screenOpen },
    { k: 'toggle', t: T.sqlOn }, { k: 'toggle', t: T.gray }, { k: 'toggle', t: T.light }, { k: 'toggle', t: T.dark },
    { k: 'unfold', t: T.algoOpen },
    { k: 'splitIn', t: X }, { k: 'ghost', t: X + 1.5 }, { k: 'splitOut', t: X + 4.6 },
    ...T.opUp.map((t, i) => ({ k: 'step', t, n: i })), ...T.opDown.map((t, i) => ({ k: 'step', t, n: 8 - i })),
    { k: 'mode', t: T.f10on }, { k: 'mode', t: T.f10off }, { k: 'mode', t: T.f6 }, { k: 'mode', t: T.f4 },
    span('charge', T.hoverA, T.hoverFire), span('scroll', T.hoverFire, T.hoverB),
    span('slide', T.moveL, T.moveL + 0.8), span('slide', T.moveR, T.moveR + 0.8),
    { k: 'page', t: T.nextQ }, { k: 'pop', t: T.folderIn }, { k: 'pop', t: T.viewerIn }, { k: 'unfold', t: T.clickMd },
    { k: 'whoosh', t: X + 22.4 }, { k: 'whoosh', t: Y }, { k: 'whoosh', t: SA.send }, { k: 'whoosh', t: T.save1 + 0.2 },
    { k: 'dip', t: I - 0.35 }, { k: 'mode', t: T.f5b },
    ...ROUNDS.flatMap(r => [span('speech', r.speak[0] + 0.1, r.speak[1]), { k: 'ask', t: r.ask }, span('stream', r.ans[0], r.ans[1])]),
    { k: 'toggle', t: T.pause },
    { k: 'cards', t: T.gridA + 0.35 }, { k: 'outro', t: C.outro },
  ].sort((a, b) => a.t - b.t)
  const timeline = { dur: DUR, fps: FPS, chapters: C, marks: MARKS, keys: keyTimes, clicks: CLICKS, cues: CUES }

  let theme = qs.get('theme') || 'dark'
  const ready = loadApp(theme).then(() => { render(0) })

  window.STAGE = { DUR, FPS, ready, frameAt, still, render, timeline, copy: { ANSWER, SHOTS } }

  // ---- 实时预览 ----
  if (!CAPTURE) {
    const seek = $('#seek'), clock = $('#clock'), pp = $('#pp')
    let t = Number(qs.get('t') || 0), playing = !qs.has('paused'), last = performance.now()
    seek.max = DUR
    pp.onclick = () => { playing = !playing; pp.textContent = playing ? '暂停' : '播放' }
    seek.oninput = () => { t = Number(seek.value) }
    addEventListener('keydown', (e) => {
      if (e.code === 'Space') { e.preventDefault(); pp.onclick() }
      if (e.code === 'ArrowRight') t = Math.min(DUR, t + (e.shiftKey ? 5 : 1))
      if (e.code === 'ArrowLeft') t = Math.max(0, t - (e.shiftKey ? 5 : 1))
    })
    ready.then(function loop() {
      const now = performance.now()
      if (playing) t = (t + (now - last) / 1000) % DUR
      last = now
      settleApp(render(t))
      seek.value = t
      clock.textContent = t.toFixed(2)
      requestAnimationFrame(loop)
    })
  }
})()
