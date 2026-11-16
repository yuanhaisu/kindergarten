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
    location /api/v1/ { auth_basic off; proxy_pass http://127.0.0.1:8081; }
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
    location = /privacy { auth_basic off; try_files $uri/index.html =404; }
    location / {
        auth_basic "Kindergarten staging";
        auth_basic_user_file /etc/nginx/htpasswd/kindergarten-staging;
        try_files $uri $uri/index.html =404;
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

test('missing template fails without a stack trace', (t) => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'nginx-auth-'))
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))

  const result = run(path.join(dir, 'missing.conf.tmpl'))

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /missing\.conf\.tmpl/)
  assert.doesNotMatch(result.stderr, /\n\s+at /)
})
