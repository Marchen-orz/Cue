"""把成品 BGM（promo/Sleek Impact.mp3）按视频章节重排，输出 promo/bgm.wav 供 music.py 作音乐床。

曲子自身是 125 BPM、每 8 小节一个乐句；视频是 120 BPM、每小节 2 秒、章节起点落在小节线上。
这里不整体拉伸，而是按小节切段，让曲子的段落落点对上章节：

    第一次进鼓（歌的第 9 小节） → 42 s  解题
    暗段（41–48 小节）          → 74 s  进入隐身
    第二次进鼓（57 小节）       → 100 s 复盘
    回落后第三次进鼓（81 小节） → 148 s 面试开始收音
    结尾和弦（92 小节）         → 178 s 片尾

接缝只裁 / 重复整 4 小节（一个和弦循环），和声不会错位；接缝处做 45 ms 的短交叉淡化。
每段用 rubberband 变速到目标时长（保持音高），速度比在 0.89–1.03 之间。

小节线位置来自对成品的分析：BPM 125.03，第一个下拍在 1.811 s。换了曲子需要重新测这两个数、
重新排 PLAN。

用法:
  python promo/arrange.py [--src "promo/Sleek Impact.mp3"]
"""
import argparse
import json
import pathlib
import subprocess

import imageio_ffmpeg
import numpy as np
from scipy.io import wavfile

HERE = pathlib.Path(__file__).resolve().parent
SR = 44100
BPM, DB0 = 125.03, 1.8112
BAR = 4 * 60 / BPM
V = 2.0                       # 视频一小节（120 BPM）
PRE, POST = 0.045, 0.005      # 接缝交叉淡化：新段提前 45 ms 淡入，旧段多留 5 ms

DUR = float(json.loads((HERE / '.out' / 'timeline.json').read_text(encoding='utf-8'))['dur'])


def bars(first, n):
    return DB0 + first * BAR, n * BAR


# (标签, 曲子起点 s, 曲子时长 s, 视频时长 s)；视频起点按顺序首尾相接
PLAN = []


def add(label, first_bar, n_bars, video_s=None):
    t0, d = bars(first_bar, n_bars)
    PLAN.append((label, t0, d, video_s if video_s is not None else n_bars * V))


PLAN.append(('前奏', 0.0, DB0, V))                       # 歌的第一个下拍之前的铺底
add('铺底 A', 1, 4)                                      # 2–10
add('铺底 A（重复）', 1, 4)                              # 10–18
add('铺底 B：加军鼓 / 踩镲', 5, 4)                       # 18–26
add('铺底 A（重复）', 1, 4)                              # 26–34
add('铺底 B：渐强', 5, 4)                                # 34–42
add('第一次进鼓', 9, 16)                                 # 42–74，带到暗段前的收束小节
add('暗段', 41, 8, 8 / 12 * 26)                          # 74–100：暗段 + 渐强共 12 小节放进 26 s，
add('渐强', 53, 4, 4 / 12 * 26)                          #         速度放慢约 11%，符合「隐身」的压暗
add('第二次进鼓 → 回落', 57, 24)                         # 100–148，连续不剪
add('第三次进鼓', 81, 8)                                 # 148–164
add('第三次进鼓（后半重复）', 85, 4)                     # 164–172
add('上升', 89, 3)                                       # 172–178，接功能宫格
PLAN.append(('结尾和弦', DB0 + 92 * BAR, 6.2, 6.0))      # 178–184


def stretch(src, t0, d, out_len):
    """取曲子 [t0 - PRE, t0 + d + POST]，变速到 out_len 秒（含接缝余量）。返回 (2, n) float64。"""
    ratio = d / out_len
    a = max(0.0, t0 - PRE)
    pre_used = t0 - a
    n_in = pre_used + d + POST
    raw = subprocess.run([
        imageio_ffmpeg.get_ffmpeg_exe(), '-loglevel', 'error', '-ss', f'{a:.5f}', '-t', f'{n_in:.5f}', '-i', str(src),
        '-af', f'rubberband=tempo={ratio:.6f}:pitch=1:transients=mixed', '-f', 'f32le', '-ac', '2', '-ar', str(SR), '-',
    ], capture_output=True, check=True).stdout
    x = np.frombuffer(raw, np.float32).reshape(-1, 2).T.astype(np.float64)
    want = int(round((pre_used + d + POST) / ratio * SR))
    x = x[:, :want] if x.shape[1] >= want else np.pad(x, ((0, 0), (0, want - x.shape[1])))
    return x, pre_used / ratio, ratio


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--src', default=str(HERE / 'Sleek Impact.mp3'))
    ap.add_argument('--out', default=str(HERE / 'bgm.wav'))
    a = ap.parse_args()

    out = np.zeros((2, int(round(DUR * SR))))
    t = 0.0
    for i, (label, t0, d, vlen) in enumerate(PLAN):
        x, pre_v, ratio = stretch(a.src, t0, d, vlen)
        post_v = POST / ratio
        fade = int(round((pre_v + post_v) * SR))
        env = np.ones(x.shape[1])
        if i > 0 and fade:
            env[:fade] = np.linspace(0, 1, fade)
        if i < len(PLAN) - 1 and fade:
            env[-fade:] = np.linspace(1, 0, fade)
        start = int(round((t - pre_v) * SR))
        lo, hi = max(0, start), min(out.shape[1], start + x.shape[1])
        out[:, lo:hi] += (x * env)[:, lo - start:hi - start]
        print(f'{t:7.2f}s  ×{ratio:.3f}  {label}')
        t += vlen
    assert abs(t - DUR) < 0.01, f'段落总长 {t:.2f}s 与片长 {DUR:.2f}s 不符'
    out /= max(np.abs(out).max(), 1e-9)
    wavfile.write(a.out, SR, (out.T * 32767 * 0.95).astype(np.int16))
    print(f'已输出 {a.out}')


if __name__ == '__main__':
    main()
