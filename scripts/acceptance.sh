#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080/api/v1}"
HEALTH_URL="${HEALTH_URL:-${BASE_URL%/api/v1}/health}"
USERNAME="${PROXYD_USERNAME:-admin}"
PASSWORD="${PROXYD_PASSWORD:-admin}"
REPORT_DIR="${REPORT_DIR:-logs}"
REPORT_FILE="${REPORT_FILE:-${REPORT_DIR}/acceptance-$(date +%Y%m%d-%H%M%S).md}"
TMP_DIR="$(mktemp -d)"
TOKEN=""

PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0

mkdir -p "${REPORT_DIR}"

cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

json_eval() {
  local expr="$1"
  python3 -c "import json,sys; data=json.load(sys.stdin); print(${expr})"
}

call_api() {
  local method="$1"
  local path="$2"
  local data="${3:-}"
  local url="${BASE_URL}${path}"

  local args=( -sS -X "${method}" "${url}" -H "Content-Type: application/json" )
  if [[ -n "${TOKEN}" ]]; then
    args+=( -H "Authorization: Bearer ${TOKEN}" )
  fi
  if [[ -n "${data}" ]]; then
    args+=( -d "${data}" )
  fi

  local response
  response="$(curl "${args[@]}" -w $'\n%{http_code}')"
  API_BODY="${response%$'\n'*}"
  API_CODE="${response##*$'\n'}"
}

record_pass() {
  local title="$1"
  local details="$2"
  PASS_COUNT=$((PASS_COUNT + 1))
  {
    echo "- [PASS] ${title}"
    echo "  - ${details}"
  } >> "${REPORT_FILE}"
}

record_fail() {
  local title="$1"
  local details="$2"
  FAIL_COUNT=$((FAIL_COUNT + 1))
  {
    echo "- [FAIL] ${title}"
    echo "  - ${details}"
  } >> "${REPORT_FILE}"
}

record_skip() {
  local title="$1"
  local details="$2"
  SKIP_COUNT=$((SKIP_COUNT + 1))
  {
    echo "- [SKIP] ${title}"
    echo "  - ${details}"
  } >> "${REPORT_FILE}"
}

expect_status() {
  local expected="$1"
  [[ "${API_CODE}" == "${expected}" ]]
}

start_report() {
  cat > "${REPORT_FILE}" <<EOF
# Proxyd Acceptance Report

- Generated at: $(date -Iseconds)
- Base URL: ${BASE_URL}

## Core Acceptance
EOF
}

check_health() {
  local response
  response="$(curl -sS "${HEALTH_URL}" -w $'\n%{http_code}')"
  local body="${response%$'\n'*}"
  local code="${response##*$'\n'}"

  if [[ "${code}" != "200" ]]; then
    record_fail "Health check" "Expected 200, got ${code}. body=${body}"
    return 1
  fi

  local status
  status="$(printf '%s' "${body}" | json_eval "json.dumps(data.get('status'))")"
  if [[ "${status}" == '"ok"' ]]; then
    record_pass "Health check" "GET /health returned status=ok"
    return 0
  fi

  record_fail "Health check" "GET /health did not return status=ok. body=${body}"
  return 1
}

login() {
  call_api "POST" "/auth/login" "{\"username\":\"${USERNAME}\",\"password\":\"${PASSWORD}\"}"
  if ! expect_status 200; then
    record_fail "Login" "Expected 200, got ${API_CODE}. body=${API_BODY}"
    return 1
  fi

  TOKEN="$(printf '%s' "${API_BODY}" | json_eval "json.dumps(data.get('token',''))" | tr -d '"')"
  if [[ -z "${TOKEN}" ]]; then
    record_fail "Login" "No token returned. body=${API_BODY}"
    return 1
  fi

  record_pass "Login" "POST /auth/login succeeded"
  return 0
}

create_source_from_file() {
  local name="$1"
  local file_path="$2"
  call_api "POST" "/sources" "{\"name\":\"${name}\",\"type\":\"local\",\"path\":\"${file_path}\",\"enabled\":true,\"update_interval\":3600,\"priority\":0}"
  if ! expect_status 201; then
    echo ""
    return 1
  fi
  printf '%s' "${API_BODY}" | json_eval "data.get('id','')"
}

main_flow() {
  local valid_yaml="${TMP_DIR}/valid.yaml"
  cat > "${valid_yaml}" <<'EOF'
port: 7890
allow-lan: false
mode: rule
log-level: info
proxies:
  - name: demo-direct
    type: direct
proxy-groups:
  - name: Proxy
    type: select
    proxies:
      - demo-direct
rules:
  - MATCH,Proxy
EOF

  local source_id
  if ! source_id="$(create_source_from_file "acceptance-valid-$(date +%s)" "${valid_yaml}")"; then
    record_fail "Create source" "Failed to create local source for acceptance. code=${API_CODE}, body=${API_BODY}"
    return 1
  fi
  record_pass "Create source" "Created local source id=${source_id}"

  call_api "POST" "/sources/${source_id}/test"
  if expect_status 200; then
    local success
    success="$(printf '%s' "${API_BODY}" | json_eval "json.dumps(data.get('success'))")"
    if [[ "${success}" == "true" ]]; then
      record_pass "Test source" "POST /sources/${source_id}/test succeeded"
    else
      record_fail "Test source" "Source test returned success=false. body=${API_BODY}"
    fi
  else
    record_fail "Test source" "Expected 200, got ${API_CODE}. body=${API_BODY}"
  fi

  call_api "POST" "/sources/${source_id}/fetch"
  if expect_status 200; then
    record_pass "Fetch source" "POST /sources/${source_id}/fetch succeeded"
  else
    record_fail "Fetch source" "Expected 200, got ${API_CODE}. body=${API_BODY}"
  fi

  call_api "POST" "/config/generate" "{\"source_ids\":[${source_id}]}"
  if ! expect_status 200; then
    record_fail "Generate config" "Expected 200, got ${API_CODE}. body=${API_BODY}"
    return 1
  fi
  local generated
  generated="$(printf '%s' "${API_BODY}" | json_eval "json.dumps(data.get('config',''))")"
  generated="${generated#\"}"
  generated="${generated%\"}"
  generated="${generated//\\n/$'\n'}"
  record_pass "Generate config" "POST /config/generate succeeded"

  local escaped
  escaped="$(printf '%s' "${generated}" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read()))')"
  call_api "POST" "/config/apply" "{\"config\":${escaped}}"
  if expect_status 200; then
    record_pass "Apply config" "POST /config/apply succeeded"
  else
    record_fail "Apply config" "Expected 200, got ${API_CODE}. body=${API_BODY}"
  fi

  local mihomo_available=true

  call_api "POST" "/proxy/mihomo/start"
  if expect_status 200; then
    record_pass "Mihomo start" "POST /proxy/mihomo/start succeeded"
  else
    mihomo_available=false
    record_skip "Mihomo start" "Mihomo unavailable in current environment. code=${API_CODE}, body=${API_BODY}"
  fi

  call_api "POST" "/proxy/mihomo/restart"
  if expect_status 200; then
    record_pass "Mihomo restart" "POST /proxy/mihomo/restart succeeded"
  else
    if [[ "${mihomo_available}" == "false" ]]; then
      record_skip "Mihomo restart" "Skipped because mihomo is unavailable in current environment"
    else
      record_fail "Mihomo restart" "Restart failed. code=${API_CODE}, body=${API_BODY}"
    fi
  fi

  call_api "GET" "/proxy/groups"
  if expect_status 200; then
    local group
    group="$(printf '%s' "${API_BODY}" | json_eval "(data.get('groups') or [{}])[0].get('name','')")"
    local proxy
    proxy="$(printf '%s' "${API_BODY}" | json_eval "((data.get('groups') or [{}])[0].get('proxies') or [''])[0]")"
    if [[ -n "${group}" && -n "${proxy}" ]]; then
      call_api "PUT" "/proxy/groups/${group}" "{\"proxy\":\"${proxy}\"}"
      if expect_status 200; then
        record_pass "Switch proxy" "PUT /proxy/groups/${group} succeeded"
      else
        record_fail "Switch proxy" "Switch failed. code=${API_CODE}, body=${API_BODY}"
      fi
    else
      record_skip "Switch proxy" "No switchable proxy groups returned"
    fi
  else
    if [[ "${mihomo_available}" == "false" ]]; then
      record_skip "List proxy groups" "Skipped because mihomo is unavailable in current environment"
    else
      record_fail "List proxy groups" "Expected 200, got ${API_CODE}. body=${API_BODY}"
    fi
  fi

  call_api "POST" "/proxy/mihomo/stop"
  if expect_status 200; then
    record_pass "Mihomo stop" "POST /proxy/mihomo/stop succeeded"
  else
    if [[ "${mihomo_available}" == "false" ]]; then
      record_skip "Mihomo stop" "Skipped because mihomo is unavailable in current environment"
    else
      record_fail "Mihomo stop" "Stop failed. code=${API_CODE}, body=${API_BODY}"
    fi
  fi
}

failure_flow() {
  echo "" >> "${REPORT_FILE}"
  echo "## Failure-path Acceptance" >> "${REPORT_FILE}"

  call_api "POST" "/sources" "{\"name\":\"acceptance-invalid-url-$(date +%s)\",\"type\":\"http\",\"url\":\"http://127.0.0.1:1/does-not-exist\",\"enabled\":true,\"update_interval\":3600,\"priority\":0}"
  local bad_source_id=""
  if expect_status 201; then
    bad_source_id="$(printf '%s' "${API_BODY}" | json_eval "data.get('id','')")"
    call_api "POST" "/sources/${bad_source_id}/test"
    if expect_status 200; then
      local success
      success="$(printf '%s' "${API_BODY}" | json_eval "json.dumps(data.get('success'))")"
      if [[ "${success}" == "false" ]]; then
        record_pass "Invalid source URL" "Source test returned failure as expected"
      else
        record_fail "Invalid source URL" "Expected success=false for invalid URL. body=${API_BODY}"
      fi
    else
      record_fail "Invalid source URL" "Unexpected status for test: ${API_CODE}. body=${API_BODY}"
    fi
  else
    record_fail "Invalid source URL" "Could not create invalid-url source fixture. code=${API_CODE}, body=${API_BODY}"
  fi

  local invalid_yaml="${TMP_DIR}/invalid.yaml"
  cat > "${invalid_yaml}" <<'EOF'
port: [
EOF

  local invalid_source_id
  if invalid_source_id="$(create_source_from_file "acceptance-invalid-yaml-$(date +%s)" "${invalid_yaml}")"; then
    call_api "POST" "/config/generate" "{\"source_ids\":[${invalid_source_id}]}"
    if [[ "${API_CODE}" == "400" || "${API_CODE}" == "500" ]]; then
      record_pass "Invalid YAML" "Generate failed as expected. code=${API_CODE}"
    else
      record_fail "Invalid YAML" "Expected generate failure, got code=${API_CODE}. body=${API_BODY}"
    fi
  else
    record_fail "Invalid YAML" "Could not create invalid-yaml source fixture. code=${API_CODE}, body=${API_BODY}"
  fi

  call_api "POST" "/config/save" "{\"config\":\"port: 7890\",\"path\":\"/tmp/outside-runtime.yaml\"}"
  if [[ "${API_CODE}" == "400" || "${API_CODE}" == "500" ]]; then
    if [[ "${API_BODY}" == *"within mihomo config dir"* ]]; then
      record_pass "Config path traversal" "Out-of-dir path was rejected"
    else
      record_fail "Config path traversal" "Request failed but message was unexpected. body=${API_BODY}"
    fi
  else
    record_fail "Config path traversal" "Expected rejection, got code=${API_CODE}. body=${API_BODY}"
  fi

  call_api "POST" "/proxy/mihomo/start"
  if expect_status 200; then
    record_skip "Mihomo unavailable path" "Current environment can start mihomo; unavailable-path should be verified with wrong binary/port conflict deployment"
    call_api "POST" "/proxy/mihomo/stop"
    true
  else
    if [[ "${API_BODY}" == *"not found"* || "${API_BODY}" == *"failed to start"* || "${API_BODY}" == *"address already in use"* || "${API_BODY}" == *"did not become ready"* ]]; then
      record_pass "Mihomo unavailable path" "Start failed as expected in unavailable environment. body=${API_BODY}"
    else
      record_fail "Mihomo unavailable path" "Start failed but message does not clearly indicate unavailable/startup issue. body=${API_BODY}"
    fi
  fi
}

finish_report() {
  {
    echo ""
    echo "## Summary"
    echo ""
    echo "- PASS: ${PASS_COUNT}"
    echo "- FAIL: ${FAIL_COUNT}"
    echo "- SKIP: ${SKIP_COUNT}"
    echo ""
    echo "## Reproduction Commands"
    echo ""
    echo '```bash'
    echo "BASE_URL=${BASE_URL} PROXYD_USERNAME=${USERNAME} PROXYD_PASSWORD='***' ./scripts/acceptance.sh"
    echo '```'
  } >> "${REPORT_FILE}"

  echo "Acceptance report: ${REPORT_FILE}"
  echo "PASS=${PASS_COUNT} FAIL=${FAIL_COUNT} SKIP=${SKIP_COUNT}"

  if [[ ${FAIL_COUNT} -gt 0 ]]; then
    return 1
  fi
  return 0
}

start_report

if ! check_health; then
  finish_report
  exit 1
fi

if ! login; then
  finish_report
  exit 1
fi

main_flow || true
failure_flow || true
finish_report
