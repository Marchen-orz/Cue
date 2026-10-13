"""为 Cue 宣传片生成音轨并混入 MP4。

时间轴取自 promo/.out/timeline.json（render.py 每次运行都会写出）：片长、章节起点、
按键 / 点击时刻和各类界面事件。改了 stage.js 的节奏后重跑 render.py，这里自动跟上。

两种配乐来源：
  1. 把成品 BGM 放到 promo/bgm.mp3（或 .wav / .m4a / .flac / .ogg）——用它做音乐床，
     只叠加本脚本合成的交互音效。120 BPM、4/4、与片长等长时卡点最准。
  2. 没有 bgm.* 时，退回纯 numpy 合成的配乐（无版权素材）。

合成配乐：120 BPM，A 小调。段落跟着章节走（见 sections()）：
  片头铺底 → 悬浮窗进鼓 → 配置（律动渐满）→ 截图解题（完整律动 + 琶音）
  → 技能 / 简历（换一组和弦，加旋律）→ 隐身（滤波压暗、半速）→ 存储复盘（律动回归）
  → 外观 / 历史 / 快捷键（旋律段）→ 面试设置（抽掉鼓）→ 面试作答（完整律动）
  → 功能宫格（上升音效 → 冲击）→ 片尾终止和弦
音效：按键、鼠标点击、快门、飞入、提示音、流式输出滴答、卡片弹出等，逐个对应时间轴上的事件。

用法:
  python promo/music.py                   # 生成音轨并输出 promo/cue-promo.mp4
  python promo/music.py --wav-only        # 只生成 promo/.out/cue-promo-audio.wav
  python promo/music.py --bgm-offset 12   # 从外部 BGM 的第 12 秒开始取
"""
import argparse
import json
import pathlib
import subprocess
import sys

import imageio_ffmpeg
import numpy as np
from scipy.io import wavfile
from scipy.signal import butter, fftconvolve, sosfilt

HERE = pathlib.Path(__file__).resolve().parent
OUT = HERE / '.out'
TIMELINE = OUT / 'timeline.json'
if not TIMELINE.exists():
    sys.exit('缺少 promo/.out/timeline.json：先启动 node promo/serve.mjs 并运行 python promo/render.py')
TL = json.loads(TIMELINE.read_text(encoding='utf-8'))
CH, CUES = TL['chapters'], TL['cues']

SR = 44100
DUR = float(TL['dur'])
N = int(SR * DUR)
BEAT = 0.5          # 120 BPM
BAR = 4 * BEAT
rng = np.random.default_rng(7)

# 两组和弦进行：(和弦, 根音)
PROGS = {
    'A': ([[57, 60, 64], [53, 57, 60], [48, 52, 55], [55, 59, 62]], [45, 41, 48, 43]),   # Am F C G
    'B': ([[53, 57, 60], [55, 59, 62], [57, 60, 64], [52, 55, 59]], [41, 43, 45, 40]),   # F G Am Em
}
# B 进行上的四小节旋律：(拍, 音高)
LEAD = [(0, 72), (1.5, 69), (2.5, 72), (3, 76), (4, 74), (5.5, 71), (6.5, 74), (7, 79),
        (8, 76), (9.5, 72), (10.5, 76), (11, 81), (12, 79), (13.5, 76), (14.5, 74), (15, 71)]

# 段落配置。kick: 1 四拍 / 0.5 半速；hats: 1 反拍 / 2 加十六分 / 0.5 轻；bass: 1 八分 / 2 长音；
# bright: 铺底滤波开度；arp: 流式输出时加琶音（'all' 为整段）；fill: 段前军鼓过门
KINDS = {
    'intro':   dict(kick=0, hats=0, clap=0, bass=0, prog='A', bright=0.45),
    'enter':   dict(kick=1, hats=0, clap=0, bass=0, prog='A', bright=0.6),
    'setup':   dict(kick=1, hats=1, clap=0, bass=1, prog='A', bright=0.75),
    'setup2':  dict(kick=1, hats=2, clap=0, bass=1, prog='A', bright=0.85),
    'build':   dict(kick=1, hats=2, clap=1, bass=1, prog='A', bright=1.0),
    'full':    dict(kick=1, hats=2, clap=1, bass=1, prog='A', bright=1.0, arp=True, fill=True),
    'light':   dict(kick=1, hats=1, clap=0, bass=1, prog='B', bright=0.7, lead=True),
    'stealth': dict(kick=0.5, hats=0.5, clap=0, bass=2, prog='A', bright=0.08),
    'melody':  dict(kick=1, hats=2, clap=1, bass=1, prog='B', bright=0.85, lead=True),
    'break':   dict(kick=0, hats=0.5, clap=0, bass=2, prog='B', bright=0.35),
    'grid':    dict(kick=1, hats=2, clap=1, bass=1, prog='A', bright=1.0, arp='all'),
    'outro':   dict(kick=0, hats=0, clap=0, bass=0, prog='A', bright=1.0),
}


def snap(t):
    return round(t / BAR) * BAR


def sections():
    """[(起, 止, 段落配置)]，起止都落在小节线上。"""
    live = next((c['t'] for c in CUES if c['k'] == 'mode' and c['t'] > CH['interview']), CH['interview'] + 10)   # 面试开始收音
    plan = [
        (0, 'intro'), (CH['bar'], 'enter'), (CH['setup'], 'setup'), (TL['marks']['model'], 'setup2'),
        (CH['shots'], 'build'), (CH['solve'], 'full'), (CH['skills'], 'light'), (CH['stealth'], 'stealth'),
        (CH['review'], 'full'), (CH['history'], 'melody'), (CH['interview'], 'break'), (live, 'full'),
        (CH['grid'], 'grid'), (CH['outro'], 'outro'),
    ]
    starts = [snap(t) for t, _ in plan] + [DUR]
    return [(a, b, dict(KINDS[k], kind=k)) for a, b, (_, k) in zip(starts, starts[1:], plan)]


SECTIONS = sections()


def spans(k):
    return [(c['t'], c['t'] + c.get('d', 0)) for c in CUES if c['k'] == k]


# ---------- 基础工具 ----------
def hz(m):
    return 440.0 * 2 ** ((m - 69) / 12)


def tt(d):
    return np.arange(int(d * SR)) / SR


def filt(x, kind, f, order=2):
    return sosfilt(butter(order, f, kind, fs=SR, output='sos'), x)


def saw(f, d):
    return 2 * ((f * tt(d) + rng.random()) % 1) - 1


def sweep(f0, f1, d, curve=4.0):
    """指数扫频正弦。"""
    t = tt(d)
    f = f1 + (f0 - f1) * np.exp(-t * curve / d)
    return np.sin(2 * np.pi * np.cumsum(f) / SR)


def ad(n_or_sig, a, d_tau):
    """快起、指数衰减包络，作用于信号。"""
    sig = n_or_sig
    t = np.arange(len(sig)) / SR
    return sig * np.minimum(1, t / max(a, 1e-4)) * np.exp(-t / d_tau)


def noise(d):
    return rng.uniform(-1, 1, int(d * SR))


def bus():
    return np.zeros((2, N))


def add(b, t0, sig, g=1.0, pan=0.0):
    i = int(t0 * SR)
    if i >= N or i + len(sig) <= 0:
        return
    s = sig[:N - i] * g
    th = (pan + 1) * np.pi / 4
    b[0, i:i + len(s)] += s * np.cos(th) * np.sqrt(2)
    b[1, i:i + len(s)] += s * np.sin(th) * np.sqrt(2)


def curve(points):
    xs, ys = zip(*points)
    return np.interp(np.arange(N) / SR, xs, ys)


def in_any(t, spans):
    return any(a <= t < b for a, b in spans)


# ---------- 乐器 ----------
def kick(g=1.0):
    return ad(sweep(160, 45, 0.45, 5), 0.002, 0.16) * g


def hat(d=0.05):
    return ad(filt(noise(d), 'high', 7000), 0.001, d / 4)


def clap():
    s = np.zeros(int(0.25 * SR))
    for k, off in enumerate((0, 0.011, 0.022)):
        b = ad(filt(noise(0.2), 'band', [900, 3500]), 0.001, 0.012 if k < 2 else 0.07)
        i = int(off * SR)
        s[i:i + len(b)] += b[:len(s) - i]
    return s


def bass_note(m, d):
    s = filt(saw(hz(m), d) + 0.5 * np.sin(2 * np.pi * hz(m - 12) * tt(d)), 'low', 520)
    return ad(s, 0.004, d * 0.6)


def pluck(m, d=0.3):
    s = filt(saw(hz(m), d) * 0.6 + np.sign(np.sin(2 * np.pi * hz(m) * tt(d))) * 0.25, 'low', 3200)
    return ad(s, 0.002, 0.09)


def bell(m, d=1.2, tau=0.35):
    t = tt(d)
    f = hz(m)
    s = np.sin(2 * np.pi * f * t) + 0.35 * np.sin(2 * np.pi * f * 2.76 * t) * np.exp(-t / 0.12) \
        + 0.18 * np.sin(2 * np.pi * f * 5.4 * t) * np.exp(-t / 0.05)
    return ad(s, 0.002, tau)


def whoosh(d, up=True):
    n = noise(d)
    lo, hi = filt(n, 'band', [250, 1200]), filt(n, 'band', [1800, 8000])
    k = np.linspace(0, 1, len(n)) if up else np.linspace(1, 0, len(n))
    env = np.sin(np.pi * np.linspace(0, 1, len(n))) ** 1.6
    return (lo * (1 - k) + hi * k) * env


def riser(d):
    n = noise(d)
    k = np.linspace(0, 1, len(n)) ** 2
    s = filt(n, 'band', [400, 1500]) * (1 - k) + filt(n, 'high', 3000) * k
    tone = np.sin(2 * np.pi * np.cumsum(np.linspace(200, 900, len(n))) / SR) * 0.15
    return (s + tone) * k


def impact(g=1.0):
    boom = ad(sweep(110, 32, 1.6, 3), 0.002, 0.5)
    crash = ad(filt(noise(1.2), 'high', 2500), 0.001, 0.28) * 0.35
    body = ad(filt(noise(0.5), 'low', 900), 0.001, 0.06) * 0.8
    s = boom.copy()
    s[:len(crash)] += crash
    s[:len(body)] += body
    return s * g


def key_click():
    s = ad(filt(noise(0.05), 'high', 2500), 0.0005, 0.004) * 0.9
    s += ad(filt(noise(0.05), 'band', [700, 2200]), 0.0005, 0.015) * 0.6
    s += ad(np.sin(2 * np.pi * 190 * tt(0.05)), 0.001, 0.02) * 0.6
    return s


def mouse_click():
    s = ad(filt(noise(0.03), 'high', 3500), 0.0003, 0.003)
    s += ad(np.sin(2 * np.pi * 1700 * tt(0.03)), 0.0005, 0.006) * 0.4
    return s


def blip(f0, f1, d=0.16):
    return ad(sweep(f0, f1, d, 6), 0.003, d / 3)


def tick():
    return ad(filt(noise(0.012), 'high', 4500), 0.0003, 0.0025)


# ---------- 配乐 ----------
def compose_music():
    drums, bass, pad, arp, lead = bus(), bus(), bus(), bus(), bus()
    padsig = np.zeros(N)
    sc = np.ones(N)
    streams = spans('stream')

    for a, b, o in SECTIONS:
        chords, roots = PROGS[o['prog']]
        if o['kind'] == 'outro':     # 终止在 Am
            d = DUR - a
            t = tt(d)
            env = np.minimum(1, t / 0.35) * np.minimum(1, (d - t) / 0.45)
            i = int(a * SR)
            for m in (57, 60, 64, 69):
                for det in (-0.08, 0, 0.08):
                    padsig[i:] += (saw(hz(m + det), d) * env * 0.09)[:N - i]
            continue

        # 鼓组 + 侧链
        for t0 in np.arange(a, b, BEAT):
            k = round((t0 - a) / BEAT)
            if o['kick'] == 1 or (o['kick'] == 0.5 and k % 4 == 0):
                add(drums, t0, kick(0.95 if o['kick'] == 1 else 0.6))
            if o['kick'] == 1:
                i = int(t0 * SR)
                e = 1 - 0.55 * np.exp(-tt(0.4) / 0.09)
                sc[i:i + len(e)] = np.minimum(sc[i:i + len(e)], e[:N - i])
            if o['hats'] >= 1:
                add(drums, t0 + BEAT / 2, hat(), 0.32, 0.25)
            if o['hats'] == 2:
                add(drums, t0 + BEAT / 4, hat(0.03), 0.1, -0.3)
            if o['hats'] == 0.5:
                add(drums, t0 + BEAT / 2, hat(0.03), 0.12, 0.3)
            if o['clap'] and k % 2 == 1:
                add(drums, t0, clap(), 0.4)
        if o.get('fill'):            # 段前一秒的军鼓滚奏
            for k in range(8):
                add(drums, a - 1 + k * BEAT / 4, clap(), 0.12 + 0.04 * k, (k % 2 - 0.5) * 0.4)

        for n, t0 in enumerate(np.arange(a, b, BAR)):
            ch, root = chords[n % 4], roots[n % 4]
            # 贝斯
            if o['bass'] == 1:
                for k8 in range(8):
                    add(bass, t0 + k8 * BEAT / 2, bass_note(root + (12 if k8 % 4 == 3 else 0), 0.24), 0.55)
            elif o['bass'] == 2:
                add(bass, t0, bass_note(root - 12, BAR) * 1.4, 0.2)
            # 铺底和弦
            d = BAR + 0.5
            t = tt(d)
            env = np.minimum(1, t / 0.35) * np.minimum(1, (d - t) / 0.45)
            i = int(t0 * SR)
            for m in ch + [ch[0] + 12]:
                for det in (-0.08, 0, 0.08):
                    sig = saw(hz(m + det), d) * env * 0.09
                    padsig[i:i + len(sig)] += sig[:N - i]
            # 琶音（16 分音符，乒乓延迟）
            tones = [c + 12 for c in ch] + [ch[0] + 24]
            for k16 in range(16):
                t1 = t0 + k16 * BEAT / 4
                if not (o.get('arp') == 'all' or (o.get('arp') and in_any(t1, streams))):
                    continue
                g = 0.22 if k16 % 4 == 0 else 0.15
                pan = 0.35 if k16 % 2 else -0.35
                sig = pluck(tones[[0, 1, 2, 3, 2, 1, 0, 2][k16 % 8]])
                add(arp, t1, sig, g, pan)
                add(arp, t1 + 0.375, sig, g * 0.35, -pan)
                add(arp, t1 + 0.75, sig, g * 0.14, pan)
            # 旋律
            if o.get('lead'):
                for beat, m in LEAD:
                    if beat // 4 == n % 4:
                        sig = bell(m, 1.1, 0.28)
                        t1 = t0 + (beat % 4) * BEAT
                        add(lead, t1, sig, 0.085, 0.2)
                        add(lead, t1 + 0.375, sig, 0.03, -0.4)

    # 铺底滤波与音量随段落开合
    pts, vol, prev = [(0, 0)], [(0, 0), (2.5, 0.9), (4, 0.72)], 0
    for a, b, o in SECTIONS:
        pts += [(a + 0.01, prev), (a + 0.7, o['bright'])]
        prev = o['bright']
        if a > 4:
            v = 0.85 if o['kind'] == 'stealth' else 1.0 if o['kind'] == 'outro' else 0.72
            vol += [(a, vol[-1][1]), (a + 0.5, v)]
    mixk = curve(pts + [(DUR, prev)])
    dark, bright = filt(padsig, 'low', 700), filt(padsig, 'low', 3200)
    p = (dark * (1 - mixk) + bright * mixk) * curve(vol + [(DUR, vol[-1][1])])
    pad[0] += p
    pad[1] += np.roll(p, int(0.012 * SR))   # 轻微拉宽

    pad *= sc
    bass *= sc
    dry = drums * 0.4 + bass * 0.5 + pad * 0.75 + arp * 1.3 + lead
    send = pad * 0.35 + arp * 0.5 + lead * 0.6 + drums * 0.06
    return dry, send


# ---------- 音效（逐个对应时间轴上的事件） ----------
def compose_sfx(with_hits=True):
    """with_hits=False 时省掉冲击 / 上升等「音乐性」音效，避免和外部 BGM 打架。"""
    sfx = bus()

    def stream(a, b, pan=0.5):
        t0 = a
        while t0 < b:
            add(sfx, t0, tick(), 0.1 + 0.05 * rng.random(), pan + 0.2 * rng.random())
            t0 += 0.045 + 0.05 * rng.random()

    def chime(t0, notes, g=0.14, pan=0.5):
        for i, m in enumerate(notes):
            add(sfx, t0 + i * 0.07, bell(m, 1.0, 0.3), g * (1 - 0.25 * i), pan)

    def typing(t0, d, g=0.2, pan=0.4):
        for t1 in np.arange(t0, t0 + d, 0.075):
            add(sfx, t1 + 0.012 * rng.random(), key_click(), g + 0.06 * rng.random(), pan)

    def pulses(t0, d, m=93):
        for t1 in np.arange(t0, t0 + d, 0.33):
            add(sfx, t1, bell(m, 0.3, 0.05), 0.04, 0.6)

    shutter = np.zeros(int(0.12 * SR))                        # 快门：双响
    kc = key_click()
    shutter[:len(kc)] += kc * 0.9
    j = int(0.055 * SR)
    shutter[j:j + len(kc)] += kc

    # 片头（Logo 动画，不在事件表里）
    if with_hits:
        add(sfx, 0.2, impact(0.6))
    add(sfx, 0.55, whoosh(0.75), 0.16)                        # C 形弧线画出
    chime(1.3, (93, 100), 0.16, 0.2)                          # 圆点「叮」
    add(sfx, 3.5, whoosh(0.9), 0.35)                          # 进入桌面

    for t0 in TL['keys']:
        add(sfx, t0, key_click(), 0.5)
    for t0 in TL['clicks']:
        add(sfx, t0, mouse_click(), 0.45, 0.3)

    for c in CUES:
        k, t0, d = c['k'], c['t'], c.get('d', 0)
        if k == 'drop':                                       # 工具条落下
            add(sfx, t0, whoosh(0.5, up=False), 0.3, 0.5)
            add(sfx, t0 + 0.55, blip(500, 900, 0.12), 0.18, 0.5)
        elif k == 'expand':                                   # 面板展开
            add(sfx, t0, whoosh(0.45), 0.25, 0.5)
        elif k == 'open':                                     # 设置弹出
            add(sfx, t0 + 0.05, whoosh(0.4), 0.2, 0.4)
        elif k == 'close':
            add(sfx, t0 + 0.05, whoosh(0.35, up=False), 0.16, 0.4)
        elif k == 'toast':
            add(sfx, t0, bell(86, 0.8, 0.2), 0.1, 0.5)
        elif k == 'warn':
            chime(t0, (83, 78), 0.12)
        elif k == 'type':                                     # 在输入框里打字
            typing(t0, d)
        elif k == 'typeDesk':                                 # 穿透后在下层窗口打字
            typing(t0, d, 0.16, -0.3)
        elif k in ('spin', 'think'):                          # 等待中的轻脉冲
            pulses(t0, d, 93 if k == 'think' else 89)
        elif k == 'whoosh':                                   # 镜头推拉
            add(sfx, t0, whoosh(0.8), 0.28)
        elif k == 'drag':                                     # 框选拖拽
            add(sfx, t0, whoosh(d), 0.08, -0.4)
        elif k == 'shutter':
            add(sfx, t0, shutter, 0.6)
            add(sfx, t0 + 0.05, ad(filt(noise(0.12), 'band', [1400, 6500]), 0.001, 0.03), 0.35)
            add(sfx, t0 + 0.05, whoosh(0.65), 0.4, 0.2)       # 缩略图飞入截图坞
            add(sfx, t0 + 0.63, blip(700, 1300, 0.09), 0.3, 0.6)
        elif k == 'remove':
            add(sfx, t0, blip(1100, 600, 0.12), 0.22, 0.5)
        elif k == 'send':
            add(sfx, t0 + 0.1, blip(600, 1200, 0.14), 0.2, 0.5)
        elif k == 'stream':                                   # 流式输出
            add(sfx, t0, bell(81, 1.0, 0.25), 0.12, 0.6)
            stream(t0, t0 + d)
        elif k == 'done':
            chime(t0, (84, 88, 91), 0.12)
        elif k == 'pop':                                      # 系统窗口 / 下拉弹出
            add(sfx, t0, blip(520, 880, 0.1), 0.16, 0.2)
            add(sfx, t0, whoosh(0.3), 0.14, 0.2)
        elif k == 'toggle':
            add(sfx, t0, blip(700, 1300, 0.1), 0.25, 0.5)
        elif k == 'unfold':
            add(sfx, t0 + 0.05, whoosh(0.35, up=False), 0.15, 0.4)
        elif k == 'splitIn':                                  # 拉开成分屏
            add(sfx, t0, whoosh(0.9, up=False), 0.35)
        elif k == 'ghost':                                    # 「Cue 不在画面里」
            add(sfx, t0, blip(1300, 420, 0.25), 0.3, 0.5)
        elif k == 'splitOut':
            add(sfx, t0, whoosh(0.9), 0.3)
        elif k == 'step':                                     # 透明度逐级调节
            add(sfx, t0, bell(96 - c.get('n', 0), 0.2, 0.04), 0.06, 0.4)
        elif k == 'mode':                                     # 模式开关
            chime(t0 + 0.02, (88, 95), 0.13)
        elif k == 'charge':                                   # 悬停蓄力
            add(sfx, t0, sweep(900, 320, d, 3)[::-1] * np.linspace(0, 1, int(d * SR)) ** 1.5, 0.05, 0.5)
            add(sfx, t0 + d, blip(900, 1500, 0.12), 0.2, 0.5)
        elif k == 'scroll':
            for t1 in np.arange(t0, t0 + d, 0.11):
                add(sfx, t1, tick(), 0.07, 0.5)
        elif k == 'slide':                                    # 窗口平移
            add(sfx, t0, whoosh(d), 0.12, 0.3)
        elif k == 'page':
            add(sfx, t0, whoosh(0.3), 0.2, -0.4)
        elif k == 'dip':                                      # 切到视频面试
            add(sfx, t0, whoosh(0.7), 0.35)
        elif k == 'speech':                                   # 实时转写逐字出现
            for t1 in np.arange(t0, t0 + d, 0.1):
                add(sfx, t1, bell(100, 0.08, 0.015), 0.035, 0.5)
        elif k == 'ask':                                      # 判定为问题
            add(sfx, t0, bell(88, 0.8, 0.25), 0.12, 0.6)
        elif k == 'cards':                                    # 功能宫格：卡片逐个弹出
            if with_hits:
                add(sfx, t0 - 1.5, riser(1.3), 0.3)
                add(sfx, t0 - 0.2, impact(0.55))
            else:
                add(sfx, t0 - 0.5, whoosh(0.6), 0.3)
            for i, m in enumerate((69, 72, 74, 76, 79, 81, 84, 86)):
                add(sfx, t0 + 0.15 + i * 0.09, bell(m, 0.6, 0.12), 0.16, (i % 4 - 1.5) * 0.35)
        elif k == 'outro':                                    # 片尾 Logo
            add(sfx, t0 - 0.2, whoosh(0.55), 0.3)
            if with_hits:
                add(sfx, t0 + 0.3, impact(0.8))
            chime(t0 + 1.2, (93, 100), 0.14, 0.0)             # 圆点
            for i, m in enumerate((81, 88, 93)):              # 快捷键 / 仓库地址 / 署名依次升起
                add(sfx, t0 + 1.3 + i * 0.4, bell(m, 1.4, 0.45), 0.08, (i - 1) * 0.5)
    return sfx


def reverb_ir(d=2.2):
    t = tt(d)
    ir = np.stack([filt(noise(d), 'low', 6000), filt(noise(d), 'low', 6000)]) * np.exp(-t / 0.55)
    ir[:, :int(0.015 * SR)] = 0
    return ir / np.abs(ir).sum(axis=1, keepdims=True) * 40


def find_bgm():
    for ext in ('mp3', 'wav', 'm4a', 'flac', 'ogg'):
        f = HERE / f'bgm.{ext}'
        if f.exists():
            return f
    return None


def load_bgm(path, offset):
    raw = subprocess.run([
        imageio_ffmpeg.get_ffmpeg_exe(), '-loglevel', 'error', '-ss', str(offset), '-i', str(path),
        '-t', str(DUR), '-f', 'f32le', '-ac', '2', '-ar', str(SR), '-',
    ], capture_output=True, check=True).stdout
    x = np.frombuffer(raw, np.float32).reshape(-1, 2).T.astype(np.float64)
    if x.shape[1] < N - SR:
        print(f'注意：{path.name} 从 {offset}s 起只有 {x.shape[1] / SR:.1f}s，不足 {DUR:.0f}s，尾部将无配乐')
    bed = np.zeros((2, N))
    bed[:, :min(N, x.shape[1])] = x[:, :N]
    return bed / max(np.abs(bed).max(), 1e-9) * 0.7


def mix(bgm=None, offset=0.0):
    ir = reverb_ir()
    if bgm:
        sfx = compose_sfx(with_hits=False)
        wet = np.stack([fftconvolve(sfx[c] * 0.25, ir[c])[:N] for c in range(2)])
        out = load_bgm(bgm, offset) + sfx * 0.8 + wet * 0.3
    else:
        dry, send = compose_music()
        sfx = compose_sfx()
        send = send + sfx * 0.25
        wet = np.stack([fftconvolve(send[c], ir[c])[:N] for c in range(2)])
        out = dry + sfx + wet * 0.35
    out = np.tanh(out * 1.2) / np.tanh(1.2)
    fade = np.ones(N)
    fi = int(0.05 * SR)
    fade[:fi] = np.linspace(0, 1, fi)
    fo = int(1.6 * SR)
    fade[-fo:] = np.linspace(1, 0, fo) ** 2
    out *= fade
    return out / np.abs(out).max() * 0.89


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--wav-only', action='store_true')
    ap.add_argument('--no-bgm', action='store_true', help='即使存在 bgm.* 也用合成配乐')
    ap.add_argument('--bgm-offset', type=float, default=0.0, help='从外部 BGM 的第几秒开始取')
    ap.add_argument('--video', default=str(OUT / 'cue-promo-silent.mp4'))
    ap.add_argument('--out', default=str(HERE / 'cue-promo.mp4'))
    a = ap.parse_args()

    bgm = None if a.no_bgm else find_bgm()
    print(f'配乐：{bgm.name if bgm else "合成"}')
    OUT.mkdir(exist_ok=True)
    wav = OUT / 'cue-promo-audio.wav'
    wavfile.write(wav, SR, (mix(bgm, a.bgm_offset).T * 32767).astype(np.int16))
    print(f'已输出 {wav}')
    if a.wav_only:
        return
    subprocess.run([
        imageio_ffmpeg.get_ffmpeg_exe(), '-y', '-loglevel', 'error',
        '-i', a.video, '-i', str(wav), '-map', '0:v', '-map', '1:a',
        '-c:v', 'copy', '-c:a', 'aac', '-b:a', '192k', '-shortest', '-movflags', '+faststart', a.out,
    ], check=True)
    print(f'已输出 {a.out}')


if __name__ == '__main__':
    main()
