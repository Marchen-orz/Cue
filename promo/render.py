"""逐帧渲染 Cue 宣传片，并导出 README 用图。

先启动素材服务器（真实前端 + 假后端）：
  node promo/serve.mjs

然后：
  python promo/render.py --prep           # 生成片中用到的截图素材 promo/shots/（首次渲染会自动执行）
  python promo/render.py --jobs 4         # 输出 promo/.out/cue-promo-silent.mp4（1920x1080, 30fps，无声），分 4 段并行
  python promo/render.py --frames 5 12.5  # 仅导出指定秒数的静帧到 promo/.out/frames/
  python promo/render.py --stills         # 导出 README 用图到 assets/

每次运行都会把舞台的时间轴写到 promo/.out/timeline.json，配乐脚本 music.py 据此对齐音效与段落。
"""
import argparse
import base64
import io
import json
import os
import pathlib
import subprocess
import sys

import imageio_ffmpeg
from PIL import Image
from playwright.sync_api import sync_playwright

HERE = pathlib.Path(__file__).resolve().parent
ROOT = HERE.parent
OUT = HERE / '.out'
SHOTS = HERE / 'shots'
ASSETS = ROOT / 'assets'
URL = os.environ.get('PROMO_URL', 'http://127.0.0.1:5199/__promo/index.html')
CHROME = os.path.expanduser('~/AppData/Local/ms-playwright/chromium-1228/chrome-win64/chrome.exe')

# 镜头：[tx, ty, scale]，与 stage.js 一致；None 表示沿用时间线上的镜头
MID = [-384, -72, 1.2]
ZOOM = [-864, -171, 1.45]
REGION = {'x': 22, 'y': 152, 'width': 678, 'height': 648}   # 片中框选的截图区域，与 stage.js 的 SEL 一致
PREROLL = 4.0   # 从中途开始渲染时先空跑的秒数：让弹窗、下拉、过渡动画落到正确状态

# README 用图。mark 是 stage.js 里 MARKS 的取景时刻，state / solve / settings 是在那一刻基础上的补丁。
# crop='app' 只截 Cue 窗口、crop='modal' 只截设置弹窗（都带一圈边距），
# crop=[x, y, w, h] 截画面中的指定区域，否则截整个画面。
Q2 = {'exam': {'q': 2, 'scroll': 0}}
STILLS = {
    'banner': dict(mark='history', cam=[-70, 50, 1], bare=True, banner='zh', crop=[0, 140, 1920, 800], solve={'scroll': 0.42}),
    'banner-en': dict(mark='history', cam=[-70, 50, 1], bare=True, banner='en', crop=[0, 140, 1920, 800], solve={'scroll': 0.42}),
    'shot-solve': dict(mark='history', cam=MID, state=Q2, solve={'scroll': 0.42}),
    'shot-interview': dict(mark='interview', cam=MID),
    'shot-stealth': dict(mark='split', cam=None, crop=[0, 100, 1920, 660]),
    'shot-through': dict(mark='through', cam=None, cursor=True),
    'shot-review': dict(mark='review', windows=True),
    'shot-toolbar': dict(mark='bar', cam=[-1344, -102, 1.7], crop='app', bare=True, state={'collapsed': True, 'status': '已连接'}),
    'shot-welcome': dict(mark='ready', cam=ZOOM, crop='app', bare=True),
    'shot-skills': dict(mark='skills', cam=ZOOM, crop='modal', bare=True),
    'shot-interview-settings': dict(mark='ivSettings', cam=ZOOM, crop='modal', bare=True, settings={'scroll': 0}),
    'shot-model': dict(mark='model', cam=ZOOM, crop='modal', bare=True, theme='light'),
    'shot-screenshot': dict(mark='shotSettings', cam=ZOOM, crop='modal', bare=True, theme='light'),
    'shot-shortcuts': dict(mark='shortcuts', cam=ZOOM, crop='modal', bare=True, theme='light'),
    'shot-light': dict(mark='history', cam=ZOOM, crop='app', bare=True, theme='light', solve={'scroll': 0}),
}


def launch(p):
    return p.chromium.launch(executable_path=CHROME if os.path.exists(CHROME) else None)


def open_page(p, scale=1, query='', quiet=False):
    browser = launch(p)
    page = browser.new_page(viewport={'width': 1920, 'height': 1080}, device_scale_factor=scale)
    page.on('pageerror', lambda e: print(f'[pageerror] {e}', file=sys.stderr))
    page.on('console', lambda m: m.type == 'error' and not quiet and print(f'[console] {m.text}', file=sys.stderr))
    page.on('response', lambda r: r.status >= 400 and not quiet and print(f'[{r.status}] {r.url}', file=sys.stderr))
    page.goto(f'{URL}?capture=1{query}')
    page.wait_for_function('window.STAGE !== undefined')
    page.evaluate('STAGE.ready')
    OUT.mkdir(exist_ok=True)
    (OUT / 'timeline.json').write_text(json.dumps(page.evaluate('STAGE.timeline'), ensure_ascii=False, indent=1), encoding='utf-8')
    return browser, page


def prep(p):
    """生成片中「截图」的结果：把笔试页的题面按片中的框选区域截下来，再用真实的导出功能生成一张答案图。"""
    SHOTS.mkdir(exist_ok=True)
    browser, page = open_page(p, quiet=True)   # 素材还没生成，舞台页预加载它们时的 404 不用报

    def grab(name, state, clip=None, gray=False):
        page.evaluate('o => STAGE.still(o)', {'t': 5, 'noapp': True, 'state': state})
        img = Image.open(io.BytesIO(page.screenshot(clip=clip))).convert('L' if gray else 'RGB')
        img.save(SHOTS / f'{name}.jpg', quality=88)
        print(f'shots/{name}.jpg', flush=True)

    for name, scroll in (('q2-a', 0), ('q2-b', 0.5), ('q2-c', 1)):
        grab(name, {'exam': {'q': 2, 'scroll': scroll}}, REGION)
    grab('q2-a-gray', {'exam': {'q': 2, 'scroll': 0}}, REGION, gray=True)
    grab('q3', {'exam': {'q': 3, 'scroll': 0}}, REGION)
    grab('screen1-gray', {'exam': {'q': 2, 'scroll': 0}}, gray=True)
    grab('screen2-gray', {'meet': True}, gray=True)
    copy = page.evaluate('STAGE.copy')
    browser.close()

    # 「导出为图片」：在不带虚拟时钟的应用页里跑真实的 exportImage，假后端把结果留在 __cue.exported。
    # 用浅色主题——片中的导出发生在切到浅色之后（导出卡片里的代码配色跟随当前主题）
    browser = launch(p)
    page = browser.new_page(viewport={'width': 1280, 'height': 900})
    page.on('pageerror', lambda e: print(f'[pageerror] {e}', file=sys.stderr))
    page.goto(URL.split('/__promo/')[0] + '/?theme=light')
    page.wait_for_function("window.__cue && window.__cue.store('solution') && document.querySelector('.top-bar')")
    data = page.evaluate('''async ({ answer, shot }) => {
        const sol = window.__cue.store('solution')
        sol.history = [{ time: '14:30:43', rounds: [{ userScreenshot: shot, thinking: '', thinkingStatus: '', aiResponse: answer, error: null }] }]
        sol.activeHistoryIndex = 0
        await new Promise(r => setTimeout(r, 300))
        await sol.exportImage(0)
        return window.__cue.exported
    }''', {'answer': copy['ANSWER'], 'shot': copy['SHOTS'][0]})
    browser.close()
    if not data:
        sys.exit('导出答案图失败：exportImage 没有产出图片')
    card = Image.open(io.BytesIO(base64.b64decode(data.split(',', 1)[-1])))
    card.save(SHOTS / 'export.png', optimize=True)
    print(f'shots/export.png {card.size}', flush=True)


def frames(times, preroll):
    out = OUT / 'frames'
    out.mkdir(parents=True, exist_ok=True)
    with sync_playwright() as p:
        browser, page = open_page(p)
        last = None
        for t in times:
            # 界面状态依赖前序帧：从目标时刻前几秒起逐帧推进
            start = max(0, t - preroll) if last is None or t < last or t - last > preroll else last
            n = int(round((t - start) * 30))
            for i in range(n + 1):
                page.evaluate('t => STAGE.frameAt(t)', start + i / 30)
            page.screenshot(path=str(out / f't{t:06.2f}.jpg'), type='jpeg', quality=88)
            last = t
            print(f't={t}', flush=True)
        browser.close()


def stills(names):
    ASSETS.mkdir(exist_ok=True)
    with sync_playwright() as p:
        browser, page = open_page(p, scale=2)
        # 同主题的图排在一起，少重载几次应用
        for name in sorted(names, key=lambda n: STILLS[n].get('theme', 'dark')):
            opts = dict(STILLS[name])
            crop = opts.pop('crop', None)
            rects = page.evaluate('o => STAGE.still(o)', opts)
            clip = None
            if isinstance(crop, list):
                clip = dict(zip(('x', 'y', 'width', 'height'), crop))
            elif crop:
                rect, m = rects[crop], 32
                x, y = max(0, rect['x'] - m), max(0, rect['y'] - m)
                clip = {'x': x, 'y': y, 'width': min(rect['x'] + rect['w'] + m, 1920) - x,
                        'height': min(rect['y'] + rect['h'] + m, 1080) - y}
            if name.startswith('banner'):  # 大面积渐变，JPEG 比 PNG 小得多
                page.screenshot(path=str(ASSETS / f'{name}.jpg'), clip=clip, type='jpeg', quality=90)
            else:
                page.screenshot(path=str(ASSETS / f'{name}.png'), clip=clip)
            print(name, flush=True)
        browser.close()


def video(fps, dst, start, end, tag=''):
    dst.parent.mkdir(parents=True, exist_ok=True)
    ff = subprocess.Popen([
        imageio_ffmpeg.get_ffmpeg_exe(), '-y', '-loglevel', 'error',
        '-f', 'image2pipe', '-framerate', str(fps), '-c:v', 'mjpeg', '-i', '-',
        '-vf', 'scale=in_range=pc:out_range=tv,format=yuv420p',
        '-c:v', 'libx264', '-preset', 'slow', '-crf', '17',
        '-movflags', '+faststart', str(dst),
    ], stdin=subprocess.PIPE)
    with sync_playwright() as p:
        browser, page = open_page(p)
        duration = page.evaluate('STAGE.DUR')
        end = min(end or duration, duration)
        first, total = int(round(start * fps)), int(round(end * fps))
        for i in range(max(0, first - int(PREROLL * fps)), first):
            page.evaluate('t => STAGE.frameAt(t)', i / fps)
        for i in range(first, total):
            page.evaluate('t => STAGE.frameAt(t)', i / fps)
            ff.stdin.write(page.screenshot(type='jpeg', quality=96))
            if i % (fps * 2) == 0:
                print(f'{tag}{i // fps:>3}/{int(end)}s', flush=True)
        browser.close()
    ff.stdin.close()
    ff.wait()
    print(f'{tag}已输出 {dst}', flush=True)


def video_parallel(fps, dst, jobs):
    """按章节边界切成几段并行渲染，再无损拼接。切点选在画面平静处（见 timeline 的 chapters）。"""
    tl = json.loads((OUT / 'timeline.json').read_text(encoding='utf-8'))
    ch = tl['chapters']
    cuts = [ch[k] for k in ('shots', 'solve', 'skills', 'stealth', 'review', 'look', 'interview')]
    # 贪心地把片长分成 jobs 段：每段尽量接近平均长度
    bounds, want = [0], tl['dur'] / jobs
    for c in cuts:
        if len(bounds) < jobs and c - bounds[-1] >= want * 0.75:
            bounds.append(c)
    bounds.append(tl['dur'])
    parts, procs = [], []
    for i, (a, b) in enumerate(zip(bounds, bounds[1:])):
        part = OUT / f'part{i}.mp4'
        parts.append(part)
        procs.append(subprocess.Popen([sys.executable, '-X', 'utf8', __file__, '--fps', str(fps), '--start', str(a), '--end', str(b),
                                       '--out', str(part), '--tag', f'[{i + 1}/{len(bounds) - 1}] ']))
    if any(pr.wait() for pr in procs):
        sys.exit('有分段渲染失败')
    lst = OUT / 'parts.txt'
    lst.write_text(''.join(f"file '{p.name}'\n" for p in parts), encoding='utf-8')
    subprocess.run([imageio_ffmpeg.get_ffmpeg_exe(), '-y', '-loglevel', 'error', '-f', 'concat', '-safe', '0', '-i', str(lst),
                    '-c', 'copy', '-movflags', '+faststart', str(dst)], check=True)
    for part in parts:
        part.unlink()
    lst.unlink()
    print(f'已输出 {dst}')


if __name__ == '__main__':
    ap = argparse.ArgumentParser()
    ap.add_argument('--frames', nargs='*', type=float)
    ap.add_argument('--stills', nargs='*')
    ap.add_argument('--prep', action='store_true')
    ap.add_argument('--fps', type=int, default=30)
    ap.add_argument('--start', type=float, default=0)
    ap.add_argument('--end', type=float, default=0)
    ap.add_argument('--jobs', type=int, default=1, help='分成几段并行渲染')
    ap.add_argument('--preroll', type=float, default=PREROLL, help='--frames 取帧前先空跑的秒数')
    ap.add_argument('--out', default=str(OUT / 'cue-promo-silent.mp4'))
    ap.add_argument('--tag', default='', help=argparse.SUPPRESS)
    a = ap.parse_args()

    if a.prep or not (SHOTS / 'export.png').exists():
        with sync_playwright() as p:
            prep(p)
        if a.prep:
            sys.exit(0)
    if a.frames is not None:
        frames(a.frames, a.preroll)
    elif a.stills is not None:
        stills(a.stills or list(STILLS))
    elif a.jobs > 1:
        with sync_playwright() as p:
            open_page(p)[0].close()   # 刷新 timeline.json
        video_parallel(a.fps, pathlib.Path(a.out), a.jobs)
    else:
        video(a.fps, pathlib.Path(a.out), a.start, a.end, a.tag)
