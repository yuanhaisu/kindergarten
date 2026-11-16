#!/usr/bin/env bash
# 部署脚本黑盒测试运行器。
#
# 自动发现同目录下的 test-*.sh 并逐个执行；新增用例只需新增文件，不改本脚本。
# 测试通过 PATH stub 隔离 docker / curl / aliyun / ln / sleep，不访问网络，
# 不触碰真实部署目录与真实云资源。

# 被 sh 等其他 shell 拉起时先给出明确错误，而不是在 BASH_SOURCE 上崩掉。
if [ -z "${BASH_VERSION:-}" ]; then
  echo "!! run.sh requires bash (invoke: bash $0)" >&2
  exit 2
fi

set -uo pipefail

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
scripts_dir=$(cd "$here/.." && pwd)

# 运行器与用例依赖的宿主命令；缺失时明确失败，不静默跳过。
# jq / tar 是被测脚本的真实依赖，测试不替换它们。
required=(bash jq tar mktemp readlink seq grep awk ln)
missing=()
for cmd in "${required[@]}"; do
  command -v "$cmd" >/dev/null 2>&1 || missing+=("$cmd")
done
if [ "${#missing[@]}" -gt 0 ]; then
  echo "!! missing required commands: ${missing[*]}" >&2
  echo "   install them and re-run; the tests themselves never hit the network." >&2
  exit 2
fi

# 语法先于行为：三个部署脚本与用例自身都要通过 bash -n。
shopt -s nullglob
syntax_targets=("$scripts_dir"/*.sh "$here"/*.sh)
shopt -u nullglob
for target in "${syntax_targets[@]}"; do
  if ! bash -n "$target"; then
    echo "!! syntax error: $target" >&2
    exit 1
  fi
done

shopt -s nullglob
tests=("$here"/test-*.sh)
shopt -u nullglob
if [ "${#tests[@]}" -eq 0 ]; then
  echo "!! no test-*.sh files found in $here" >&2
  exit 1
fi

failed=0
for test_file in "${tests[@]}"; do
  name=$(basename "$test_file")
  echo
  echo "==> $name"
  if bash "$test_file"; then
    echo "PASS $name"
  else
    echo "FAIL $name" >&2
    failed=$((failed + 1))
  fi
done

echo
if [ "$failed" -gt 0 ]; then
  echo "!! $failed of ${#tests[@]} test files failed" >&2
  exit 1
fi
echo "==> all ${#tests[@]} test files passed"
