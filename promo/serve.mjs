// 宣传素材开发服务器：用 Vite 跑真实前端，并注入假的 Wails 后端（mock.js）。
// 不改动 frontend/ 下任何源码——mock 通过 transformIndexHtml 注入。
//
//   node promo/serve.mjs            # http://localhost:5199/__promo/index.html 实时预览宣传片
//   http://localhost:5199/          # 带假后端的应用本体（可直接点着玩）
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import fs from 'node:fs'

const here = path.dirname(fileURLToPath(import.meta.url))
const frontend = path.resolve(here, '../frontend')
const require = createRequire(path.join(frontend, 'package.json'))
const { createServer } = require('vite')
const vue = require('@vitejs/plugin-vue')

const PORT = Number(process.env.PROMO_PORT || 5199)
const MIME = {
  '.html': 'text/html; charset=utf-8', '.js': 'text/javascript; charset=utf-8',
  '.mjs': 'text/javascript; charset=utf-8', '.css': 'text/css; charset=utf-8',
  '.json': 'application/json; charset=utf-8', '.svg': 'image/svg+xml',
  '.png': 'image/png', '.jpg': 'image/jpeg', '.wav': 'audio/wav',
  '.mp3': 'audio/mpeg', '.m4a': 'audio/mp4', '.mp4': 'video/mp4',
}

const promoPlugin = {
  name: 'cue-promo',
  transformIndexHtml(html) {
    // 必须先于 main.js 执行：应用启动即调用 window.go / window.runtime
    return html.replace('<head>', '<head>\n    <script src="/__promo/mock.js"></script>')
  },
  configureServer(server) {
    server.middlewares.use('/__promo', (req, res, next) => {
      const rel = decodeURIComponent((req.url || '/').split('?')[0])
      const file = path.join(here, rel === '/' ? 'index.html' : rel)
      if (!file.startsWith(here) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) return next()
      res.setHeader('Content-Type', MIME[path.extname(file).toLowerCase()] || 'application/octet-stream')
      res.setHeader('Cache-Control', 'no-store')
      fs.createReadStream(file).pipe(res)
    })
  },
}

const server = await createServer({
  root: frontend,
  configFile: false,
  plugins: [vue(), promoPlugin],
  server: { port: PORT, strictPort: true, host: '127.0.0.1' },
  logLevel: 'warn',
})
await server.listen()
console.log(`cue-promo ready: http://127.0.0.1:${PORT}/__promo/index.html`)
