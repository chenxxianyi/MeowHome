import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const issues = []

const apiConfig = readFileSync(resolve(root, 'src/api/config.ts'), 'utf8')
const origin = apiConfig.match(/API_ORIGIN\s*=\s*['"]([^'"]+)['"]/)?.[1] ?? ''
if (!origin.startsWith('https://')) issues.push(`API_ORIGIN 仍不是 HTTPS 正式地址：${origin || '未找到'}`)
if (/localhost|127\.0\.0\.1|192\.168\.|10\./.test(origin)) issues.push('API_ORIGIN 仍指向本机或局域网')

const manifest = readFileSync(resolve(root, 'src/manifest.json'), 'utf8')
const manifestAppId = manifest.match(/"mp-weixin"\s*:\s*\{[\s\S]*?"appid"\s*:\s*"([^"]*)"/)?.[1] ?? ''
if (!manifestAppId) issues.push('manifest.json 尚未填写 mp-weixin 正式 appid')
if (/"urlCheck"\s*:\s*false/.test(manifest)) issues.push('manifest.json 仍关闭合法域名校验')

const project = JSON.parse(readFileSync(resolve(root, 'project.config.json'), 'utf8'))
if (!project.appid) issues.push('project.config.json 尚未填写 appid')
if (manifestAppId && project.appid && manifestAppId !== project.appid)
  issues.push('manifest 与 project.config 的 appid 不一致')

const pages = JSON.parse(readFileSync(resolve(root, 'src/pages.json'), 'utf8')).pages.map((page) => page.path)
for (const testPage of ['pages/tag-test/index', 'pages/icon-test/index']) {
  if (pages.includes(testPage)) issues.push(`发布前需移除验证页：${testPage}`)
}

const output = resolve(root, 'dist/build/mp-weixin')
function files(path) {
  return readdirSync(path).flatMap((name) => {
    const child = resolve(path, name)
    return statSync(child).isDirectory() ? files(child) : [child]
  })
}
if (!existsSync(output)) {
  issues.push('尚未生成 dist/build/mp-weixin，请先执行 build:mp-weixin')
} else {
  const bytes = files(output).reduce((sum, file) => sum + statSync(file).size, 0)
  if (bytes >= 2 * 1024 * 1024) issues.push(`主包达到 ${(bytes / 1024 / 1024).toFixed(2)} MB，超过 2 MB`)
}

if (issues.length) {
  console.error(`上线检查未通过（${issues.length} 项）：`)
  for (const issue of issues) console.error(`- ${issue}`)
  process.exit(1)
}

console.log('上线检查通过：HTTPS、appid、合法域名校验、验证页与包体检查均合格。')
