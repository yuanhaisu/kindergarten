import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const SCRIPT = path.join(repoRoot, 'scripts/check-nginx-staging-auth.mjs')
const REAL_TEMPLATE = path.join(repoRoot, 'deploy/nginx/kindergarten.conf.tmpl')

/**
 * 最小合法模板：覆盖脚本要求的全部 server 块。
 * 负例在这个基线上去掉 / 加入指令，避免测试跟真实模板的排版耦合。
 */
function validTemplate() {
  return `# fixture
limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;

server {
    listen 80;
    server_name api.\${DOMAIN} admin.\${DOMAIN} test-api.\${DOMAIN} test-admin.\${DOMAIN} \${DOMAIN} www.\${DOMAIN} test-www.\${DOMAIN};

    location ^~ /.well-known/acme-challenge/ {
        root /var/www/acme;
        default_type text/plain;
        try_files $uri =404;
    }
    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl;
    server_name api.\${DOMAIN};
    location /api/v1/ { proxy_pass http://127.0.0.1:8080; }
}

server {
    listen 443 ssl;
    server_name test-api.\${DOMAIN};
    location /api/v1/ { proxy_pass http://127.0.0.1:8081; }
}

server {
    listen 443 ssl;
    server_name admin.\${DOMAIN};
    location / { try_files $uri $uri/ /index.html; }
}

server {
    listen 443 ssl;
    server_name test-admin.\${DOMAIN};
    location /_nuxt/ {
        auth_basic "Kindergarten staging";
        auth_basic_user_file /etc/nginx/htpasswd/kindergarten-staging;
        try_files $uri =404;
    }
    location / {
        auth_basic "Kindergarten staging";
        auth_basic_user_file /etc/nginx/htpasswd/kindergarten-staging;
        try_files $uri $uri/ /index.html;
    }
    location /api/v1/ {
        auth_basic off;
        proxy_pass http://127.0.0.1:8081;
    }
    location = /health {
        auth_basic off;
        proxy_pass http://127.0.0.1:8081;
    }
    location = /ready {
        auth_basic off;
        proxy_pass http://127.0.0.1:8081;
    }
}

server {
    listen 443 ssl;
    server_name \${DOMAIN};
    return 301 https://www.\${DOMAIN}$request_uri;
}

server {
    listen 443 ssl;
    server_name www.\${DOMAIN};
    location / { try_files $uri $uri/index.html =404; }
}

server {
    listen 443 ssl;
    server_name test-www.\${DOMAIN};
    location = /privacy {
        auth_basic off;
        try_files $uri/index.html =404;
    }
    location = /terms {
        auth_basic off;
        try_files $uri/index.html =404;
    }
    location = /404.html {
        auth_basic off;
        try_files $uri =404;
    }
    location ~* ^/(screens|brand)/ {
        auth_basic off;
        try_files $uri =404;
    }
    location ~* \\.(?:css|js|mjs|json|webp|png|jpe?g|gif|svg|ico|woff2?|ttf|xml|txt|webmanifest)$ {
        auth_basic off;
        try_files $uri =404;
    }
    location ^~ /_nuxt/ {
        auth_basic off;
        try_files $uri =404;
    }
    location / {
        auth_basic "Kindergarten staging";
        auth_basic_user_file /etc/nginx/htpasswd/kindergarten-staging;
        rewrite ^/(.+)/$ /$1 permanent;
        try_files $uri $uri/index.html $uri/ =404;
    }
}
`
}

function writeFixture(t, content) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'nginx-auth-'))
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const file = path.join(dir, 'kindergarten.conf.tmpl')
  fs.writeFileSync(file, content)
  return file
}

function run(target) {
  return spawnSync(process.execPath, [SCRIPT, target], { encoding: 'utf8' })
}

const AUTH_OFF_LINE = '        auth_basic off;\n'
const STATIC_ASSETS_HEADER =
  '    location ~* \\.(?:css|js|mjs|json|webp|png|jpe?g|gif|svg|ico|woff2?|ttf|xml|txt|webmanifest)$ {'

/** 去掉某个 location 里的 auth_basic off;，模拟豁免被误删。 */
function withoutExemption(template, header) {
  const target = `${header}\n${AUTH_OFF_LINE}`
  assert.ok(template.includes(target), `fixture 缺少豁免：${header}`)
  return template.replace(target, `${header}\n`)
}

/** 整个 location 块连豁免一起删掉，模拟资源回落到带门禁的 location /。 */
function withoutLocationBlock(template, header) {
  const start = template.indexOf(header)
  assert.notEqual(start, -1, `fixture 缺少 location：${header}`)
  const end = template.indexOf('\n    }', start)
  assert.notEqual(end, -1, `fixture location 未闭合：${header}`)
  return template.slice(0, start) + template.slice(end + '\n    }\n'.length)
}

function removeLastOccurrence(template, text) {
  const index = template.lastIndexOf(text)
  assert.notEqual(index, -1, `fixture 缺少文本：${text}`)
  return template.slice(0, index) + template.slice(index + text.length)
}

test('real template passes', () => {
  const result = run(REAL_TEMPLATE)

  assert.equal(result.status, 0, result.stderr)
  assert.match(result.stdout, /OK/)
})

test('baseline fixture passes', (t) => {
  const result = run(writeFixture(t, validTemplate()))

  assert.equal(result.status, 0, result.stderr)
})

test('test-admin missing auth_basic fails', (t) => {
  const fixture = writeFixture(
    t,
    validTemplate().replace(
      /server \{\n {4}listen 443 ssl;\n {4}server_name test-admin\.\$\{DOMAIN\};\n[\s\S]*?\n\}\n/,
      'server {\n    listen 443 ssl;\n    server_name test-admin.${DOMAIN};\n    location / { try_files $uri $uri/ /index.html; }\n}\n',
    ),
  )

  const result = run(fixture)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /test-admin/)
  assert.match(result.stderr, /auth_basic/)
})

test('test-www missing auth_basic fails', (t) => {
  const fixture = writeFixture(
    t,
    validTemplate().replace(
      /server \{\n {4}listen 443 ssl;\n {4}server_name test-www\.\$\{DOMAIN\};\n[\s\S]*?\n\}\n/,
      'server {\n    listen 443 ssl;\n    server_name test-www.${DOMAIN};\n    location / { try_files $uri =404; }\n}\n',
    ),
  )

  const result = run(fixture)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /test-www/)
  assert.match(result.stderr, /auth_basic/)
})

test('auth_basic on production admin fails', (t) => {
  const fixture = writeFixture(
    t,
    validTemplate().replace(
      'server_name admin.${DOMAIN};\n    location / {',
      'server_name admin.${DOMAIN};\n    auth_basic "leak";\n    auth_basic_user_file /etc/nginx/htpasswd/x;\n    location / {',
    ),
  )

  const result = run(fixture)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /admin/)
  assert.match(result.stderr, /不应/)
})

test('auth_basic on production www fails', (t) => {
  const fixture = writeFixture(
    t,
    validTemplate().replace(
      'server_name www.${DOMAIN};\n    location / {',
      'server_name www.${DOMAIN};\n    auth_basic "leak";\n    auth_basic_user_file /etc/nginx/htpasswd/x;\n    location / {',
    ),
  )

  const result = run(fixture)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /www/)
})

test('port 80 block without ACME location fails', (t) => {
  const fixture = writeFixture(
    t,
    validTemplate().replace(/ {4}location \^~ \/\.well-known\/acme-challenge\/ \{[\s\S]*?\n {4}\}\n/, ''),
  )

  const result = run(fixture)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /acme/i)
})

test('port 80 block with auth_basic fails', (t) => {
  const fixture = writeFixture(
    t,
    validTemplate().replace(
      'listen 80;',
      'listen 80;\n    auth_basic "leak";\n    auth_basic_user_file /etc/nginx/htpasswd/x;',
    ),
  )

  const result = run(fixture)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /80/)
})

test('test-admin /api/v1/ without auth_basic off fails', (t) => {
  const fixture = writeFixture(t, withoutExemption(validTemplate(), '    location /api/v1/ {'))

  const result = run(fixture)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /test-admin/)
  assert.match(result.stderr, /auth_basic off/)
})

test('test-admin /health without auth_basic off fails', (t) => {
  const fixture = writeFixture(t, withoutExemption(validTemplate(), '    location = /health {'))

  const result = run(fixture)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /test-admin/)
  assert.match(result.stderr, /auth_basic off/)
})

test('test-admin /ready without auth_basic off fails', (t) => {
  const fixture = writeFixture(t, withoutExemption(validTemplate(), '    location = /ready {'))

  const result = run(fixture)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /test-admin/)
  assert.match(result.stderr, /auth_basic off/)
})

test('test-www /privacy without auth_basic off fails', (t) => {
  const fixture = writeFixture(t, withoutExemption(validTemplate(), '    location = /privacy {'))

  const result = run(fixture)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /test-www/)
  assert.match(result.stderr, /auth_basic off/)
})

test('test-www /terms without auth_basic off fails', (t) => {
  const fixture = writeFixture(t, withoutExemption(validTemplate(), '    location = /terms {'))

  const result = run(fixture)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /test-www/)
  assert.match(result.stderr, /auth_basic off/)
})

test('test-www /404.html without auth_basic off fails', (t) => {
  const fixture = writeFixture(t, withoutExemption(validTemplate(), '    location = /404.html {'))

  const result = run(fixture)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /test-www/)
  assert.match(result.stderr, /auth_basic off/)
})

test('test-www /_nuxt/ location without auth_basic off fails', (t) => {
  // nginx 的 ^~ 优先于 ~*，这段匿名服务 /_nuxt/* 资源；豁免丢了同样会让政策页 401。
  const fixture = writeFixture(t, withoutExemption(validTemplate(), '    location ^~ /_nuxt/ {'))

  const result = run(fixture)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /test-www/)
  assert.match(result.stderr, /_nuxt/)
  assert.match(result.stderr, /auth_basic off/)
})

test('test-www static asset location without auth_basic off fails', (t) => {
  const fixture = writeFixture(t, withoutExemption(validTemplate(), STATIC_ASSETS_HEADER))

  const result = run(fixture)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /test-www/)
  assert.match(result.stderr, /静态资源/)
})

test('test-www static asset location with an invalid regex reports the regex error', (t) => {
  const fixture = writeFixture(
    t,
    validTemplate().replace(STATIC_ASSETS_HEADER, '    location ~* \\.css( {'),
  )

  const result = run(fixture)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /test-www/)
  // 必须点名出错的 location 与正则错误，而不是笼统的「缺少静态扩展名 location」。
  assert.match(result.stderr, /~\* \\\.css\(/)
  assert.match(result.stderr, /无法编译/)
  assert.doesNotMatch(result.stderr, /缺少覆盖静态扩展名/)
})

test('test-www without a static asset location fails', (t) => {
  const fixture = writeFixture(t, withoutLocationBlock(validTemplate(), STATIC_ASSETS_HEADER))

  const result = run(fixture)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /test-www/)
  assert.match(result.stderr, /静态扩展名|静态资源/)
})

test('test-www location / without clean URL rewrite fails', (t) => {
  const fixture = writeFixture(
    t,
    validTemplate().replace('        rewrite ^/(.+)/$ /$1 permanent;\n', ''),
  )

  const result = run(fixture)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /test-www/)
  assert.match(result.stderr, /rewrite/)
})

test('test-www location / without directory try_files fails', (t) => {
  const fixture = writeFixture(
    t,
    validTemplate().replace('        try_files $uri $uri/index.html $uri/ =404;\n', ''),
  )

  const result = run(fixture)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /test-www/)
  assert.match(result.stderr, /try_files/)
})

test('test-www location / without auth_basic_user_file fails', (t) => {
  const fixture = writeFixture(
    t,
    removeLastOccurrence(
      validTemplate(),
      '        auth_basic_user_file /etc/nginx/htpasswd/kindergarten-staging;\n',
    ),
  )

  const result = run(fixture)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /test-www/)
  assert.match(result.stderr, /auth_basic_user_file/)
})

test('missing template fails without a stack trace', (t) => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'nginx-auth-'))
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))

  const result = run(path.join(dir, 'missing.conf.tmpl'))

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /missing\.conf\.tmpl/)
  assert.doesNotMatch(result.stderr, /\n\s+at /)
})
