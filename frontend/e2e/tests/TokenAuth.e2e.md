# Token Authentication E2E Test Plan

## Test Purpose
Verify that the token-based authentication flow works end-to-end:
- Backend rejects unauthenticated requests with 401
- Backend accepts requests with valid Bearer token
- Frontend setup page requires both Server URL and Token
- Frontend automatically carries token on all API requests
- Frontend shows clear auth error when token is invalid
- Settings page allows updating token

## Prerequisites
1. Anchor Server running on http://localhost:17421 with token auth enabled
2. Vite dev server running on http://localhost:1420
3. Server API token known (check `docker logs anchor-server | grep "API Token"`)

---

## Test 1: Backend Public Route (No Token Required)

**Navigation:** None (curl)

**Commands:**
```bash
curl -s -o /dev/null -w "%{http_code}" http://localhost:17421/health
```

**Expected Result:** HTTP 200

---

## Test 2: Backend Protected Route Without Token

**Navigation:** None (curl)

**Commands:**
```bash
curl -s -o /dev/null -w "%{http_code}" http://localhost:17421/projects
```

**Expected Result:** HTTP 401

---

## Test 3: Backend Protected Route With Valid Token

**Navigation:** None (curl)

**Commands:**
```bash
curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer <VALID_TOKEN>" \
  http://localhost:17421/projects
```

**Expected Result:** HTTP 200

---

## Test 4: Backend Protected Route With Wrong Token

**Navigation:** None (curl)

**Commands:**
```bash
curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer wrong-token" \
  http://localhost:17421/projects
```

**Expected Result:** HTTP 401

---

## Test 5: Frontend Setup Page — Empty Token Validation

**Navigation:** `agent-browser open http://localhost:1420`
**Pre-step:** `agent-browser eval "localStorage.clear()"`
**Pre-step:** `agent-browser reload`

**Commands:**
1. `agent-browser fill @e3 "http://localhost:17421"`
2. `agent-browser fill @e5 ""` (leave token empty)
3. `agent-browser click @e4`

**Expected Result:**
- Page stays on setup screen
- Error message visible: "请输入 API Token"

---

## Test 6: Frontend Setup Page — Wrong Token

**Navigation:** `agent-browser open http://localhost:1420`
**Pre-step:** `agent-browser eval "localStorage.clear()"`
**Pre-step:** `agent-browser reload`

**Commands:**
1. `agent-browser fill @e3 "http://localhost:17421"`
2. `agent-browser fill @e5 "wrong-token"`
3. `agent-browser click @e4`

**Expected Result:**
- Page stays on setup screen
- Error message visible: "Token 无效，请检查输入的 API Token"

---

## Test 7: Frontend Setup Page — Valid Token

**Navigation:** `agent-browser open http://localhost:1420`
**Pre-step:** `agent-browser eval "localStorage.clear()"`
**Pre-step:** `agent-browser reload`

**Commands:**
1. `agent-browser fill @e3 "http://localhost:17421"`
2. `agent-browser fill @e5 "<VALID_TOKEN>"`
3. `agent-browser click @e4`
4. `agent-browser wait --load networkidle`
5. `agent-browser snapshot -i`

**Expected Result:**
- Navigates to Dashboard
- No auth error messages visible
- Dashboard stats load correctly (总项目数, 活跃扫描, etc.)

---

## Test 8: Authenticated Pages Load Without Errors

**Precondition:** Test 7 passed (user is authenticated)

**Commands:**
1. `agent-browser click @e4` (Projects)
2. `agent-browser wait --load networkidle`
3. `agent-browser snapshot -i`

**Expected Result:**
- Projects page loads
- No "认证失败" or "请求超时" error messages

**Commands:**
1. `agent-browser click @e5` (Workers)
2. `agent-browser wait --load networkidle`
3. `agent-browser snapshot -i`

**Expected Result:**
- Workers page loads
- No auth error messages

---

## Test 9: Settings Page Token Display and Update

**Precondition:** Test 7 passed

**Navigation:** `agent-browser click @e6` (Settings)

**Commands:**
1. `agent-browser snapshot -i`

**Expected Result:**
- Token section visible with masked token (e.g., "abcd...wxyz")
- Token input field present

**Commands:**
1. `agent-browser fill @e13 "new-valid-token"`
2. `agent-browser click @e9` (Save)
3. Wait for page reload
4. `agent-browser snapshot -i`

**Expected Result:**
- Page reloads
- New token is saved (check localStorage or observe auth behavior)

---

## Test 10: Global 401 Error Handling

**Precondition:** User is logged in with valid token

**Commands:**
1. `agent-browser eval "localStorage.setItem('anchor_api_token', 'expired-token')"`
2. `agent-browser reload`
3. `agent-browser wait 3000`
4. `agent-browser snapshot -i`

**Expected Result:**
- Dashboard shows "加载数据失败：认证失败，请检查 API Token" error banner
- Toast shows "认证失败：认证失败，请检查 API Token"

---

## Cleanup

After tests:
```bash
agent-browser close
```
