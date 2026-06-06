# shellcheck shell=bash
# 加载 functional-test.env（供 functional-test-*.sh source）

load_functional_test_env() {
  local repo_root="${1:-}"
  if [[ -z "$repo_root" ]]; then
    repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
  fi

  local candidates=()
  if [[ -n "${FUNCTIONAL_TEST_ENV:-}" ]]; then
    candidates+=("$FUNCTIONAL_TEST_ENV")
  fi
  candidates+=(
    "$repo_root/functional-test.env"
    "$repo_root/scripts/functional-test.env"
    "$repo_root/.env.functional-test"
  )

  local f
  for f in "${candidates[@]}"; do
    if [[ -f "$f" ]]; then
      set -a
      # shellcheck disable=SC1090
      source "$f"
      set +a
      echo "[env] loaded ${f}"
      return 0
    fi
  done
  return 1
}
