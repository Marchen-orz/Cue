"""为宣传片合成配乐与交互音效（纯 numpy 合成，无版权素材），并混入 MP4。

配乐：120 BPM，A 小调 Am–F–C–G 循环，段落与画面对齐：
  0–4s   片头：铺底 + Logo 冲击
  4–22s  截图 / 解答：鼓组、贝斯逐步进入，解答时加入琶音
  22–28s 隐身 / 穿透：滤波压暗、半速，营造“潜行”感
  28–34s 面试模式：律动回归
  34–40s 提供商 / 片尾：上升音效 → 冲击 → 终止和弦
音效：按键、鼠标点击、快门、飞入、角标、隐藏/唤出、提示音、流式输出滴答、卡片弹出。

用法:
  python music.py              # 生成 music.wav 并混入 q-solver-promo.mp4
  python music.py --wav-only   # 只生成 music.wav
"""
import argparse
import pathlib
import subprocess

import imageio_ffmpeg
import numpy as np
from scipy.io import wavfile
from scipy.signal import butter, fftconvolve, sosfilt

HERE = pathlib.Path(__file__).resolve().parent
SR = 44100
DUR = 40.0
N = int(SR * DUR)
BEAT = 0.5          # 120 BPM
BAR = 4 * BEAT
rng = np.random.default_rng(7)

CHORDS = [[57, 60, 64], [53, 57, 60], [48, 52, 55], [55, 59, 62]]   # Am F C G
ROOTS = [45, 41, 48, 43]


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


# ---------- 编排 ----------
def compose():
    drums, bass, pad, arp, sfx = bus(), bus(), bus(), bus(), bus()

    groove = [(4, 22), (28, 34), (35, 37.5)]
    full = [(8, 22), (28, 34), (35, 37.5)]
    claps = [(13.0, 22), (30.5, 34), (35, 37.5)]
    arps = [(13.5, 22), (30.5, 34), (35, 37.5)]
    stealth = (22, 28)

    # 鼓组 + 侧链
    sc = np.ones(N)
    beats = np.arange(0, DUR, BEAT)
    for b in beats:
        if in_any(b, groove):
            add(drums, b, kick(0.95))
            i = int(b * SR)
            e = 1 - 0.55 * np.exp(-tt(0.4) / 0.09)
            sc[i:i + len(e)] = np.minimum(sc[i:i + len(e)], e[:N - i])
        elif stealth[0] <= b < stealth[1] and round(b / BEAT) % 4 == 0:
            add(drums, b, kick(0.6))
        if in_any(b, full):
            add(drums, b + BEAT / 2, hat(), 0.32, 0.25)
            add(drums, b + BEAT / 4, hat(0.03), 0.1, -0.3)
        elif stealth[0] <= b < stealth[1]:
            add(drums, b + BEAT / 2, hat(0.03), 0.12, 0.3)
        if in_any(b, claps) and round(b / BEAT) % 2 == 1:
            add(drums, b, clap(), 0.4)
    # 过门：解答前、面试前、提供商前的军鼓滚奏
    for t0 in (12.0, 27.0, 33.5):
        for k in range(8):
            add(drums, t0 + k * BEAT / 4, clap(), 0.12 + 0.04 * k, (k % 2 - 0.5) * 0.4)

    # 贝斯
    for k8 in range(int(DUR / (BEAT / 2))):
        t0 = k8 * BEAT / 2
        root = ROOTS[int(t0 // BAR) % 4]
        if in_any(t0, groove):
            m = root + (12 if k8 % 4 == 3 else 0)
            add(bass, t0, bass_note(m, 0.24), 0.55)
    for bar in range(11, 14):          # 潜行段：长音贝斯
        add(bass, bar * BAR, bass_note(ROOTS[bar % 4] - 12, BAR) * 1.4, 0.2)

    # 铺底和弦
    padsig = np.zeros(N)
    for bar in range(int(DUR / BAR)):
        notes = CHORDS[bar % 4] + [CHORDS[bar % 4][0] + 12]
        if bar >= 19:                  # 片尾终止在 Am
            notes = [57, 60, 64, 69]
        d = BAR + 0.5 if bar < 19 else 2.2
        t = tt(d)
        env = np.minimum(1, t / 0.35) * np.minimum(1, (d - t) / 0.45)
        for m in notes:
            for det in (-0.08, 0, 0.08):
                s = saw(hz(m + det), d) * env * 0.09
                i = int(bar * BAR * SR)
                padsig[i:i + len(s)] += s[:N - i]
    dark, bright = filt(padsig, 'low', 700), filt(padsig, 'low', 3200)
    mixk = curve([(0, 0), (3.4, 0.5), (8, 0.6), (13, 1), (22, 1), (22.5, 0.05), (27.5, 0.2),
                  (28.2, 0.75), (34, 1), (40, 1)])
    padv = curve([(0, 0), (2.5, 0.9), (4, 0.7), (22, 0.7), (22.5, 0.85), (28, 0.8), (34, 0.75),
                  (37.6, 0.8), (37.8, 1.0), (40, 1.0)])
    p = (dark * (1 - mixk) + bright * mixk) * padv
    pad[0] += p
    pad[1] += np.roll(p, int(0.012 * SR))   # 轻微拉宽

    # 琶音（16 分音符，乒乓延迟）
    order = [0, 1, 2, 3, 2, 1, 0, 2]
    for k16 in range(int(DUR / (BEAT / 4))):
        t0 = k16 * BEAT / 4
        if not in_any(t0, arps):
            continue
        ch = CHORDS[int(t0 // BAR) % 4]
        tones = [c + 12 for c in ch] + [ch[0] + 24]
        m = tones[order[k16 % 8]]
        g = 0.22 if k16 % 4 == 0 else 0.15
        pan = 0.35 if k16 % 2 else -0.35
        s = pluck(m)
        add(arp, t0, s, g, pan)
        add(arp, t0 + 0.375, s, g * 0.35, -pan)
        add(arp, t0 + 0.75, s, g * 0.14, pan)

    # ---------- 音效（与画面时间轴对齐） ----------
    add(sfx, 0.2, impact(0.7))                                # Logo 弹入
    for i, m in enumerate((69, 76, 81)):
        add(sfx, 0.25 + i * 0.09, bell(m + 12, 1.5, 0.5), 0.12, (i - 1) * 0.4)
    add(sfx, 3.2, whoosh(0.9), 0.35)                          # 进入桌面
    add(sfx, 4.55, whoosh(0.5, up=False), 0.3, 0.5)           # 浮窗落下
    add(sfx, 5.0, blip(500, 900, 0.12), 0.18, 0.5)

    for kt, keys in ((8.0, 1), (13.2, 2), (23.0, 1), (24.3, 1), (25.0, 1)):
        for k in range(keys):
            add(sfx, kt - 0.07 * (keys - 1 - k), key_click(), 0.55)

    add(sfx, 8.05, blip(900, 1400, 0.1), 0.12)                # 进入截图模式
    add(sfx, 9.1, mouse_click(), 0.5, -0.6)                   # 按下开始框选
    add(sfx, 9.15, whoosh(1.15), 0.08, -0.4)                  # 拖拽
    add(sfx, 10.35, mouse_click(), 0.45, -0.4)                # 松开
    shutter = np.zeros(int(0.12 * SR))                            # 快门：双响
    kc = key_click()
    shutter[:len(kc)] += kc * 0.9
    j = int(0.055 * SR)
    shutter[j:j + len(kc)] += kc
    add(sfx, 10.45, shutter, 0.6)
    add(sfx, 10.52, ad(filt(noise(0.12), 'band', [1400, 6500]), 0.001, 0.03), 0.35)
    add(sfx, 10.6, whoosh(0.85), 0.4, 0.2)                    # 缩略图飞入工具栏
    add(sfx, 11.4, blip(700, 1300, 0.09), 0.3, 0.6)           # 角标

    add(sfx, 13.35, whoosh(0.75, up=False), 0.3, 0.6)         # 面板展开
    for t0 in np.arange(13.9, 15.4, 0.33):                    # 思考中的轻脉冲
        add(sfx, t0, bell(93, 0.3, 0.05), 0.04, 0.6)
    add(sfx, 15.4, bell(81, 1.0, 0.25), 0.12, 0.6)            # 开始输出

    t0 = 15.4                                                 # 流式输出滴答声
    while t0 < 21.4:
        add(sfx, t0, tick(), 0.1 + 0.05 * rng.random(), 0.5 + 0.2 * rng.random())
        t0 += 0.045 + 0.05 * rng.random()
    t0 = 30.9
    while t0 < 33.9:
        add(sfx, t0, tick(), 0.1 + 0.05 * rng.random(), 0.5 + 0.2 * rng.random())
        t0 += 0.045 + 0.05 * rng.random()

    add(sfx, 22.15, whoosh(0.55, up=False), 0.3, 0.7)         # 共享预览滑入
    add(sfx, 23.1, blip(1300, 420, 0.22), 0.3, 0.5)           # F9 隐藏
    add(sfx, 24.4, blip(420, 1300, 0.22), 0.3, 0.5)           # F9 唤出
    add(sfx, 25.05, bell(88, 1.0, 0.3), 0.14, 0.5)            # 穿透开启
    add(sfx, 25.12, bell(95, 1.0, 0.3), 0.1, 0.5)
    add(sfx, 26.6, mouse_click(), 0.55, 0.3)                  # 穿透点击
    add(sfx, 27.45, whoosh(0.7), 0.35)                        # 切换到会议
    add(sfx, 28.35, mouse_click(), 0.5, 0.6)                  # 麦克风
    add(sfx, 28.4, bell(84, 1.0, 0.3), 0.14, 0.6)
    add(sfx, 28.5, bell(91, 1.0, 0.3), 0.1, 0.6)
    t0 = 28.9                                                 # 实时字幕逐字出现
    while t0 < 30.6:
        add(sfx, t0, bell(100, 0.08, 0.015), 0.035, 0.5)
        t0 += 0.1
    add(sfx, 30.8, bell(88, 0.8, 0.25), 0.12, 0.6)            # 识别完成

    add(sfx, 33.2, riser(1.4), 0.3)                           # 转场上升
    add(sfx, 34.6, impact(0.55))
    for i in range(8):                                        # 提供商卡片逐个弹出
        m = [69, 72, 74, 76, 79, 81, 84, 86][i]
        add(sfx, 35.0 + i * 0.12, bell(m, 0.6, 0.12), 0.16, (i % 4 - 1.5) * 0.35)
    add(sfx, 37.25, whoosh(0.55), 0.3)
    add(sfx, 37.8, impact(0.8))                               # 片尾 Logo
    for i, m in enumerate((81, 88, 93)):
        add(sfx, 38.6 + i * 0.1, bell(m, 1.4, 0.45), 0.08, (i - 1) * 0.5)   # 快捷键行
    return drums, bass, pad, arp, sfx, sc


def reverb_ir(d=2.2):
    t = tt(d)
    ir = np.stack([filt(noise(d), 'low', 6000), filt(noise(d), 'low', 6000)]) * np.exp(-t / 0.55)
    ir[:, :int(0.015 * SR)] = 0
    return ir / np.abs(ir).sum(axis=1, keepdims=True) * 40


def mix():
    drums, bass, pad, arp, sfx, sc = compose()
    pad *= sc
    bass *= sc
    ir = reverb_ir()
    send = pad * 0.35 + arp * 0.5 + sfx * 0.25 + drums * 0.06
    wet = np.stack([fftconvolve(send[c], ir[c])[:N] for c in range(2)])
    music = drums * 0.4 + bass * 0.5 + pad * 0.75 + arp * 1.3
    out = music + sfx * 1.0 + wet * 0.35
    out = np.tanh(out * 1.2) / np.tanh(1.2)
    fade = np.ones(N)
    fi = int(0.05 * SR)
    fade[:fi] = np.linspace(0, 1, fi)
    fo = int(1.4 * SR)
    fade[-fo:] = np.linspace(1, 0, fo) ** 2
    out *= fade
    out = out / np.abs(out).max() * 0.89
    return out


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--wav-only', action='store_true')
    ap.add_argument('--video', default=str(HERE / 'q-solver-promo.mp4'))
    a = ap.parse_args()

    wav = HERE / 'music.wav'
    wavfile.write(wav, SR, (mix().T * 32767).astype(np.int16))
    print(f'已输出 {wav}')
    if a.wav_only:
        return
    video = pathlib.Path(a.video)
    tmp = video.with_suffix('.tmp.mp4')
    subprocess.run([
        imageio_ffmpeg.get_ffmpeg_exe(), '-y', '-loglevel', 'error',
        '-i', str(video), '-i', str(wav), '-map', '0:v', '-map', '1:a',
        '-c:v', 'copy', '-c:a', 'aac', '-b:a', '192k', '-shortest', '-movflags', '+faststart', str(tmp),
    ], check=True)
    tmp.replace(video)
    print(f'已混入 {video}')


if __name__ == '__main__':
    main()
