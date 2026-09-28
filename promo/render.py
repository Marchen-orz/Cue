"""逐帧渲染 index.html 为 MP4。

用法:
  python render.py                 # 输出 q-solver-promo.mp4（1920x1080, 30fps）
  python render.py --stills 5 10   # 仅导出指定秒数的静帧到 stills/
"""
import argparse
import os
import pathlib
import subprocess

import imageio_ffmpeg
from playwright.sync_api import sync_playwright

HERE = pathlib.Path(__file__).resolve().parent
DURATION = 40
CHROME = os.path.expanduser('~/AppData/Local/ms-playwright/chromium-1228/chrome-win64/chrome.exe')


def open_page(p):
    browser = p.chromium.launch(executable_path=CHROME if os.path.exists(CHROME) else None)
    page = browser.new_page(viewport={'width': 1920, 'height': 1080})
    page.goto((HERE / 'index.html').as_uri() + '?capture')
    page.wait_for_function('typeof window.render === "function"')
    return browser, page


def stills(times):
    out = HERE / 'stills'
    out.mkdir(exist_ok=True)
    with sync_playwright() as p:
        browser, page = open_page(p)
        for t in times:
            page.evaluate(f'render({t})')
            page.screenshot(path=str(out / f't{t:05.1f}.png'))
        browser.close()


def video(fps, dst):
    ff = subprocess.Popen([
        imageio_ffmpeg.get_ffmpeg_exe(), '-y', '-loglevel', 'error',
        '-f', 'image2pipe', '-framerate', str(fps), '-c:v', 'mjpeg', '-i', '-',
        '-vf', 'scale=in_range=pc:out_range=tv,format=yuv420p',
        '-c:v', 'libx264', '-preset', 'slow', '-crf', '18',
        '-movflags', '+faststart', str(dst),
    ], stdin=subprocess.PIPE)
    total = DURATION * fps
    with sync_playwright() as p:
        browser, page = open_page(p)
        for i in range(total):
            page.evaluate(f'render({i / fps})')
            ff.stdin.write(page.screenshot(type='jpeg', quality=95))
            if i % fps == 0:
                print(f'\r{i // fps:>2}/{DURATION}s', end='', flush=True)
        browser.close()
    ff.stdin.close()
    ff.wait()
    print(f'\n已输出 {dst}')


if __name__ == '__main__':
    ap = argparse.ArgumentParser()
    ap.add_argument('--stills', nargs='*', type=float)
    ap.add_argument('--fps', type=int, default=30)
    ap.add_argument('--out', default=str(HERE / 'q-solver-promo.mp4'))
    a = ap.parse_args()
    if a.stills is not None:
        stills(a.stills)
    else:
        video(a.fps, a.out)
