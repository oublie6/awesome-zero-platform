import { mkdir } from 'node:fs/promises'
import { dirname } from 'node:path'
import { expect, test, type Page, type Route } from '@playwright/test'
import type { Account, AuditEvent, Me, Permission, Resource, Role } from '../src/types/admin'

const account = (id: number, displayName = `Operator ${id}`): Account => ({
  id: `00000000-0000-4000-8000-${String(id).padStart(12, '0')}`,
  username: `operator-${id}`,
  email: `operator-${id}@example.test`,
  displayName,
  status: 'active',
  createdAt: '2026-10-30T10:00:00Z',
  updatedAt: '2026-10-30T10:00:00Z',
})

const seedAccounts = Array.from({ length: 41 }, (_, index) => account(index + 1, index === 0 ? 'Alice Example' : `Operator ${index + 1}`))
const seedRoles: Role[] = [{
  code: 'platform_operator', displayName: '平台操作员', description: '日常管理', system: false,
  createdAt: '2026-10-30T10:00:00Z', updatedAt: '2026-10-30T10:00:00Z',
}]
const seedResources: Resource[] = [{
  code: 'admin.account', displayName: '账号管理', module: 'identity', pattern: '/admin/accounts/*',
  actions: ['GET', 'POST', 'PATCH', 'DELETE'], description: '管理账号', system: false,
  createdAt: '2026-10-30T10:00:00Z', updatedAt: '2026-10-30T10:00:00Z',
}]

const allowedPermissions: Permission[] = [
  ...['/admin/accounts', '/admin/accounts/*', '/admin/roles', '/admin/roles/*',
    '/admin/authorization/resources', '/admin/authorization/resources/*',
    '/admin/authorization/engine', '/admin/authorization/engine/*', '/admin/audit/events',
    '/admin/system/overview', '/admin/system/configuration', '/admin/me'].map(resource => ({ resource, action: '.*' })),
]

const me: Me = {
  account: { ...account(1, 'Platform Admin'), username: 'platform-admin' },
  roles: ['platform_super_admin'],
  permissions: allowedPermissions,
}

type MockOptions = {
  failOverviewOnce?: boolean
  failAccountsOnce?: boolean
  failBootstrapStatusOnce?: boolean
  failLogout?: boolean
  expiredPaths?: string[]
  invalidRefresh?: boolean
  permissions?: Permission[]
}

type ApiCall = { path: string; method: string; body?: Record<string, unknown>; query: URLSearchParams }

function ok(data: unknown, status = 200) {
  return { status, contentType: 'application/json', body: JSON.stringify({ code: 'OK', message: '', requestId: 'browser-test', data }) }
}

function fail(status: number, code: string, message: string) {
  return { status, contentType: 'application/json', body: JSON.stringify({ code, message, requestId: 'browser-test', data: null }) }
}

async function mockApi(page: Page, options: MockOptions = {}) {
  const calls: ApiCall[] = []
  const accounts = [...seedAccounts]
  const roles = [...seedRoles]
  const deniedOnce = new Set<string>()
  let overviewFailures = options.failOverviewOnce ? 1 : 0
  let accountFailures = options.failAccountsOnce ? 1 : 0
  let bootstrapFailures = options.failBootstrapStatusOnce ? 1 : 0
  let refreshCount = 0
  let nextAccountId = 100

  await page.route('**/api/**', async (route: Route) => {
    const request = route.request()
    const url = new URL(request.url())
    const path = url.pathname
    if (!path.startsWith('/api/')) {
      await route.continue()
      return
    }
    let body: Record<string, unknown> | undefined
    try { body = request.postDataJSON() as Record<string, unknown> | undefined } catch { /* request has no JSON body */ }
    calls.push({ path, method: request.method(), body, query: url.searchParams })

    const authorization = request.headers().authorization || ''
    if (options.expiredPaths?.includes(path) && authorization === 'Bearer expired-access' && !deniedOnce.has(path)) {
      deniedOnce.add(path)
      await route.fulfill(fail(401, 'UNAUTHORIZED', 'access token expired'))
      return
    }

    if (path === '/api/auth/login' && request.method() === 'POST') {
      if (body?.password === 'wrong-password') {
        await route.fulfill(fail(401, 'INVALID_CREDENTIALS', '账号或密码错误'))
        return
      }
      await route.fulfill(ok({ accessToken: 'test-access', refreshToken: 'test-refresh', tokenType: 'Bearer', accessExpiresAt: '2027-01-01T00:00:00Z', refreshExpiresAt: '2027-02-01T00:00:00Z', account: { id: me.account.id, displayName: me.account.displayName } }))
      return
    }
    if (path === '/api/auth/refresh' && request.method() === 'POST') {
      refreshCount += 1
      if (options.invalidRefresh || body?.refreshToken !== 'test-refresh') {
        await route.fulfill(fail(401, 'INVALID_REFRESH', 'refresh token expired'))
        return
      }
      await route.fulfill(ok({ accessToken: 'fresh-access', refreshToken: 'fresh-refresh', tokenType: 'Bearer', accessExpiresAt: '2027-01-01T00:00:00Z', refreshExpiresAt: '2027-02-01T00:00:00Z', account: { id: me.account.id, displayName: me.account.displayName } }))
      return
    }
    if (path === '/api/auth/logout') {
      if (options.failLogout) { await route.fulfill(fail(503, 'UNAVAILABLE', 'logout unavailable')); return }
      await route.fulfill(ok({ loggedOut: true })); return
    }
    if (path === '/api/admin/bootstrap/status') {
      if (bootstrapFailures > 0) { bootstrapFailures -= 1; await route.fulfill(fail(503, 'UNAVAILABLE', '初始化状态暂时不可用')); return }
      await route.fulfill(ok({ available: true })); return
    }
    if (path === '/api/admin/bootstrap' && request.method() === 'POST') { await route.fulfill(ok(account(200, String(body?.displayName || 'New Admin')), 201)); return }
    if (path === '/api/admin/me') { await route.fulfill(ok({ ...me, permissions: options.permissions || me.permissions })); return }
    if (path === '/api/admin/system/overview') {
      if (overviewFailures > 0) { overviewFailures -= 1; await route.fulfill(fail(503, 'UNAVAILABLE', '运行状态暂时不可用')); return }
      await route.fulfill(ok({ service: 'app-api', mysql: 'ready', redis: 'ready', authentication: true, authorization: true, admin: true, metrics: true, bootstrapAvailable: false, engine: { name: 'Casbin' } })); return
    }
    if (path === '/api/admin/system/configuration') {
      await route.fulfill(ok({ http: { port: 8888 }, authentication: { secretConfigured: true }, admin: { bootstrapTokenConfigured: true }, password: { iterations: 3 } })); return
    }
    if (path === '/api/admin/accounts' && request.method() === 'GET') {
      if (accountFailures > 0) { accountFailures -= 1; await route.fulfill(fail(503, 'UNAVAILABLE', '账号服务暂时不可用')); return }
      const search = (url.searchParams.get('search') || '').toLowerCase()
      const status = url.searchParams.get('status')
      const filtered = accounts.filter(item => (!search || `${item.displayName} ${item.username} ${item.email}`.toLowerCase().includes(search)) && (!status || item.status === status))
      const pageNo = Number(url.searchParams.get('page') || 1)
      const pageSize = Number(url.searchParams.get('pageSize') || 20)
      await route.fulfill(ok({ items: filtered.slice((pageNo - 1) * pageSize, pageNo * pageSize), total: filtered.length, page: pageNo, pageSize })); return
    }
    if (path === '/api/admin/accounts' && request.method() === 'POST') {
      const created = account(nextAccountId++, String(body?.displayName || 'New Account'))
      accounts.unshift({ ...created, username: String(body?.username || ''), email: String(body?.email || '') })
      await route.fulfill(ok(created, 201)); return
    }
    const accountMatch = path.match(/^\/api\/admin\/accounts\/([^/]+)(?:\/(.+))?$/)
    if (accountMatch) {
      const id = decodeURIComponent(accountMatch[1])
      const suffix = accountMatch[2] || ''
      const target = accounts.find(item => item.id === id) || accounts[0]
      if (suffix === 'sessions') { await route.fulfill(ok([{ id: 'session-with-long-id-00000000000000000000000000000000', accountId: id, createdAt: '2026-10-30T10:00:00Z', expiresAt: '2026-11-01T10:00:00Z' }])); return }
      if (suffix === 'roles' && request.method() === 'GET') { await route.fulfill(ok(['platform_operator'])); return }
      if (suffix === 'roles') { await route.fulfill(ok({ roles: body?.roles || [] })); return }
      if (suffix === 'revoke-sessions') { await route.fulfill(ok({ revoked: 1 })); return }
      if (suffix === 'reset-password') { await route.fulfill(ok({ reset: true })); return }
      if (suffix === 'disable' || suffix === 'enable') {
        const updated = { ...target, status: suffix === 'disable' ? 'disabled' : 'active' }
        const index = accounts.findIndex(item => item.id === id)
        if (index >= 0) accounts[index] = updated
        await route.fulfill(ok(updated)); return
      }
      if (request.method() === 'PATCH') {
        const updated = { ...target, ...(body || {}) } as Account
        const index = accounts.findIndex(item => item.id === id)
        if (index >= 0) accounts[index] = updated
        await route.fulfill(ok(updated)); return
      }
      if (request.method() === 'GET') { await route.fulfill(ok({ account: target, roles: ['platform_operator'] })); return }
    }
    if (path === '/api/admin/roles' && request.method() === 'GET') { await route.fulfill(ok(roles)); return }
    if (path === '/api/admin/roles' && request.method() === 'POST') {
      const created = { code: String(body?.code || 'new-role'), displayName: String(body?.displayName || 'New Role'), description: String(body?.description || ''), system: false, createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() }
      roles.push(created)
      await route.fulfill(ok(created, 201)); return
    }
    const rolePermissionMatch = path.match(/^\/api\/admin\/roles\/([^/]+)\/permissions$/)
    if (rolePermissionMatch) {
      if (request.method() === 'GET') { await route.fulfill(ok([{ resource: '/admin/accounts/*', action: 'GET' }])); return }
      await route.fulfill(ok(body?.permissions || [])); return
    }
    const roleMatch = path.match(/^\/api\/admin\/roles\/([^/]+)$/)
    if (roleMatch) {
      const code = decodeURIComponent(roleMatch[1])
      if (request.method() === 'DELETE') { await route.fulfill(ok({ deleted: true })); return }
      const target = roles.find(role => role.code === code) || roles[0]
      await route.fulfill(ok({ ...target, ...(body || {}) })); return
    }
    if (path === '/api/admin/authorization/resources' && request.method() === 'GET') { await route.fulfill(ok(seedResources)); return }
    if (path === '/api/admin/authorization/resources') { await route.fulfill(ok(body || {}, request.method() === 'POST' ? 201 : 200)); return }
    if (path.startsWith('/api/admin/authorization/resources/')) {
      await route.fulfill(ok(request.method() === 'DELETE' ? { deleted: true } : body || {})); return
    }
    if (path === '/api/admin/authorization/engine') { await route.fulfill(ok({ id: 'casbin', name: 'Casbin', version: 'v2', modelType: 'RBAC', policyTypes: ['p', 'g'], supportsRawPolicy: true, supportsModelInspection: true, supportsModelEditing: false, supportsPolicyExplanation: true, supportsBatchImport: true, supportsRoleHierarchy: true, loadedAt: '2026-10-30T10:00:00Z' })); return }
    if (path === '/api/admin/authorization/engine/model') { await route.fulfill(ok({ model: '[request_definition]\nr = sub, obj, act' })); return }
    if (path === '/api/admin/authorization/engine/policies' && request.method() === 'GET') { await route.fulfill(ok([{ ptype: 'p', values: ['platform_super_admin', '/*', '.*'] }, { ptype: 'g', values: ['admin-id', 'platform_super_admin'] }])); return }
    if (path === '/api/admin/authorization/engine/policies/validate') { await route.fulfill(ok({ valid: true })); return }
    if (path === '/api/admin/authorization/engine/policies') { await route.fulfill(ok({ saved: true, ruleCount: 2 })); return }
    if (path === '/api/admin/authorization/engine/explain') { await route.fulfill(ok({ allowed: true, subject: 'admin-id', resource: '/admin/accounts', action: 'GET', roles: ['platform_super_admin'], matchedRules: [] })); return }
    if (path === '/api/admin/audit/events') {
      const events: AuditEvent[] = [{ id: 'audit-1', action: 'account.update', resourceType: 'account', resourceId: 'uuid-00000000000000000000000000000001', outcome: 'success', requestId: 'request-00000000000000000000000000000001', createdAt: '2026-10-30T10:00:00Z' }]
      await route.fulfill(ok({ items: events, total: 1, page: 1, pageSize: 20 })); return
    }
    await route.fulfill(ok([]))
  })

  return { calls, get refreshCount() { return refreshCount } }
}

async function installSession(page: Page, accessToken = 'test-access', refreshToken = 'test-refresh') {
  await page.addInitScript(({ accessToken, refreshToken }) => {
    sessionStorage.setItem('awesome-zero-admin.tokens', JSON.stringify({ accessToken, refreshToken, accessExpiresAt: '2027-01-01T00:00:00Z', refreshExpiresAt: '2027-02-01T00:00:00Z' }))
  }, { accessToken, refreshToken })
}

async function clickAccountAction(page: Page, name: string | RegExp, action: string) {
  const row = page.getByRole('row', { name })
  const direct = row.locator('.account-row-actions > .el-button').filter({ hasText: action })
  if (await direct.count()) {
    await direct.first().click()
    return
  }
  await row.getByRole('button', { name: '更多账号操作' }).click()
  const candidates = await page.getByText(action, { exact: true }).all()
  for (const candidate of candidates) {
    if (await candidate.isVisible()) {
      await candidate.click()
      return
    }
  }
  throw new Error(`Account action ${action} did not become visible in the overflow menu`)
}

test('login validates the contract and reaches the dashboard', async ({ page }) => {
  const api = await mockApi(page)
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  await page.goto('/login')
  await page.getByLabel('身份标识').fill('platform-admin')
  await page.getByLabel('密码').fill('correct-horse-battery')
  await page.getByRole('button', { name: '建立管理会话' }).click()
  await expect(page).toHaveURL(/\/dashboard$/)
  await expect(page.getByRole('heading', { name: '运行概览' })).toBeVisible()
  expect(api.calls.find(call => call.path === '/api/auth/login')?.body).toEqual({ identifier: 'platform-admin', password: 'correct-horse-battery' })
  expect(errors).toEqual([])
})

test('expired access token refreshes and retries; concurrent stale requests rotate once', async ({ page }) => {
  const api = await mockApi(page, { expiredPaths: ['/api/admin/system/overview', '/api/admin/system/configuration'] })
  await installSession(page)
  await page.goto('/dashboard')
  await expect(page.getByRole('heading', { name: '运行概览' })).toBeVisible()
  await page.evaluate(() => {
    const current = JSON.parse(sessionStorage.getItem('awesome-zero-admin.tokens') || '{}')
    sessionStorage.setItem('awesome-zero-admin.tokens', JSON.stringify({ ...current, accessToken: 'expired-access' }))
  })
  await page.getByRole('menuitem', { name: '系统状态' }).click()
  await expect(page.getByRole('heading', { name: '系统状态' })).toBeVisible()
  await expect.poll(() => api.refreshCount).toBe(1)
  await expect.poll(() => api.calls.filter(call => call.path === '/api/admin/system/overview')).toHaveLength(3)
  await expect.poll(() => api.calls.filter(call => call.path === '/api/admin/system/configuration')).toHaveLength(2)
  const stored = await page.evaluate(() => JSON.parse(sessionStorage.getItem('awesome-zero-admin.tokens') || '{}')) as { accessToken?: string; refreshToken?: string }
  expect(stored).toMatchObject({ accessToken: 'fresh-access', refreshToken: 'fresh-refresh' })
})

test('failed refresh clears the browser session and returns to login', async ({ page }) => {
  const api = await mockApi(page, { expiredPaths: ['/api/admin/me'], invalidRefresh: true })
  await installSession(page, 'expired-access')
  await page.goto('/dashboard')
  await expect(page).toHaveURL(/\/login/)
  await expect(page.getByRole('heading', { name: '管理员登录' })).toBeVisible()
  expect(api.refreshCount).toBe(1)
  expect(await page.evaluate(() => sessionStorage.getItem('awesome-zero-admin.tokens'))).toBeNull()
})

test('logout clears the local session even when the server logout request fails', async ({ page }) => {
  const api = await mockApi(page, { failLogout: true })
  await installSession(page)
  await page.goto('/dashboard')
  await page.getByRole('button', { name: 'Platform Admin' }).click()
  await page.getByText('退出登录').click()
  await expect(page).toHaveURL(/\/login$/)
  expect(api.calls.some(call => call.path === '/api/auth/logout' && call.method === 'POST')).toBe(true)
  expect(await page.evaluate(() => sessionStorage.getItem('awesome-zero-admin.tokens'))).toBeNull()
})

test('route permissions hide links and block direct navigation for unauthorized users', async ({ page }) => {
  await mockApi(page, { permissions: [{ resource: '/admin/system/overview', action: 'GET' }, { resource: '/admin/me', action: 'GET' }] })
  await installSession(page)
  await page.goto('/dashboard')
  await expect(page.getByRole('heading', { name: '运行概览' })).toBeVisible()
  await expect(page.getByRole('menuitem', { name: '账号管理' })).toHaveCount(0)
  await page.goto('/accounts')
  await expect(page).toHaveURL(/\/403$/)
  await expect(page.getByRole('heading', { name: '当前账号无权访问此页面' })).toBeVisible()
})

test('dashboard API failure offers a working retry', async ({ page }, testInfo) => {
  const api = await mockApi(page, { failOverviewOnce: true })
  await installSession(page)
  await page.goto('/dashboard')
  await expect(page.getByText('运行状态暂时不可用')).toBeVisible()
  const errorShot = testInfo.outputPath('screenshots', 'dashboard-error-1280x720.png')
  await mkdir(dirname(errorShot), { recursive: true })
  await page.screenshot({ path: errorShot, fullPage: true })
  await page.getByRole('button', { name: '重试加载' }).click()
  await expect(page.getByText('app-api')).toBeVisible()
  expect(api.calls.filter(call => call.path === '/api/admin/system/overview')).toHaveLength(2)
})

test('account search and pagination send current filters and page', async ({ page }, testInfo) => {
  const api = await mockApi(page)
  await installSession(page)
  await page.goto('/accounts')
  const search = page.getByPlaceholder('用户名 / 邮箱 / 手机号 / 显示名')
  await search.fill('alice')
  await search.press('Enter')
  await expect.poll(() => api.calls.filter(call => call.path === '/api/admin/accounts' && call.method === 'GET').at(-1)?.query.get('search')).toBe('alice')
  await search.clear()
  await search.press('Enter')
  await page.locator('.el-pagination .btn-next').click()
  await expect.poll(() => api.calls.filter(call => call.path === '/api/admin/accounts' && call.method === 'GET').at(-1)?.query.get('page')).toBe('2')
  await search.fill('no-such-account')
  await search.press('Enter')
  await expect(page.locator('.el-table__empty-block')).toBeVisible()
  const emptyShot = testInfo.outputPath('screenshots', 'accounts-empty-1280x720.png')
  await mkdir(dirname(emptyShot), { recursive: true })
  await page.screenshot({ path: emptyShot, fullPage: true })
})

test('account create, edit, disable confirmation and detail drawer work', async ({ page }) => {
  const api = await mockApi(page)
  await installSession(page)
  await page.goto('/accounts')
  await page.getByRole('button', { name: '创建账号' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('用户名').fill('new-user')
  await dialog.getByLabel('显示名称').fill('New User')
  await dialog.getByLabel('初始密码').fill('correct-horse-battery')
  await dialog.getByRole('button', { name: '保存' }).click()
  await expect.poll(() => api.calls.some(call => call.path === '/api/admin/accounts' && call.method === 'POST' && call.body?.username === 'new-user')).toBe(true)
  const alice = page.getByRole('row', { name: /Alice Example/ })
  await alice.getByRole('button', { name: '编辑' }).click()
  await expect(page.getByRole('dialog')).toBeVisible()
  await page.getByRole('dialog').getByLabel('显示名称').fill('Alice Updated')
  await page.getByRole('dialog').getByRole('button', { name: '保存' }).click()
  await expect.poll(() => api.calls.some(call => call.method === 'PATCH' && call.path.endsWith(`/${seedAccounts[0].id}`))).toBe(true)
  await expect(page.getByText('Alice Updated')).toBeVisible()
  await clickAccountAction(page, /Alice Updated/, '禁用')
  const confirmation = page.locator('.el-message-box')
  await expect(confirmation).toBeVisible()
  await confirmation.getByRole('button', { name: '禁用' }).click()
  await expect.poll(() => api.calls.some(call => call.path.endsWith(`/${seedAccounts[0].id}/disable`))).toBe(true)
  await clickAccountAction(page, /Alice Updated/, '详情')
  await expect(page.getByRole('heading', { name: '账号详情' })).toBeVisible()
  await expect(page.getByText('platform_operator')).toBeVisible()
})

test('dangerous account-session revoke requires confirmation', async ({ page }) => {
  const api = await mockApi(page)
  await installSession(page)
  await page.goto('/accounts')
  await clickAccountAction(page, /Alice Example/, '详情')
  await page.getByRole('tab', { name: '活跃会话' }).click()
  await page.getByRole('button', { name: '撤销全部会话' }).click()
  const confirmation = page.locator('.el-message-box')
  await expect(confirmation).toBeVisible()
  await confirmation.getByRole('button', { name: '取消' }).click()
  expect(api.calls.some(call => call.path.endsWith('/revoke-sessions'))).toBe(false)
  await page.getByRole('button', { name: '撤销全部会话' }).click()
  await page.locator('.el-message-box').getByRole('button', { name: '撤销会话' }).click()
  await expect.poll(() => api.calls.some(call => call.path.endsWith('/revoke-sessions'))).toBe(true)
})

test('role permissions save, role creation and delete confirmation are handled', async ({ page }) => {
  const api = await mockApi(page)
  await installSession(page)
  await page.goto('/roles')
  await expect(page.getByRole('heading', { name: '平台操作员' })).toBeVisible()
  await page.getByText('POST', { exact: true }).click()
  await page.getByRole('button', { name: '保存权限' }).click()
  await expect.poll(() => api.calls.some(call => call.method === 'PUT' && call.path.endsWith('/permissions'))).toBe(true)
  await page.getByRole('button', { name: '创建角色' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('角色编码').fill('support_operator')
  await dialog.getByLabel('显示名称').fill('客服操作员')
  await dialog.getByRole('button', { name: '保存' }).click()
  await expect.poll(() => api.calls.some(call => call.path === '/api/admin/roles' && call.method === 'POST' && call.body?.code === 'support_operator')).toBe(true)
  await page.getByRole('button', { name: '删除' }).click()
  const confirmation = page.locator('.el-message-box')
  await expect(confirmation).toBeVisible()
  await confirmation.getByRole('button', { name: '取消' }).click()
  expect(api.calls.some(call => call.method === 'DELETE' && call.path.includes('/api/admin/roles/'))).toBe(false)
})

test('expert policy replacement can be cancelled without sending a write', async ({ page }) => {
  const api = await mockApi(page)
  await installSession(page)
  await page.goto('/authorization/engine')
  await page.getByRole('tab', { name: '原始策略' }).click()
  await page.getByRole('button', { name: '整体替换' }).click()
  const confirmation = page.locator('.el-message-box')
  await expect(confirmation).toBeVisible()
  await confirmation.getByRole('button', { name: '取消' }).click()
  expect(api.calls.some(call => call.method === 'PUT' && call.path === '/api/admin/authorization/engine/policies')).toBe(false)
})

test('bootstrap status failure has a recoverable retry', async ({ page }) => {
  const api = await mockApi(page, { failBootstrapStatusOnce: true })
  await page.goto('/bootstrap')
  await expect(page.getByText('初始化状态暂时不可用')).toBeVisible()
  await page.getByRole('button', { name: '重试检查' }).click()
  await expect(page.getByLabel('Bootstrap Token')).toBeVisible()
  expect(api.calls.filter(call => call.path === '/api/admin/bootstrap/status')).toHaveLength(2)
})

test('mobile navigation opens as a drawer and provides reachable menu items', async ({ page }) => {
  await mockApi(page)
  await installSession(page)
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/dashboard')
  await page.getByRole('button', { name: '打开导航菜单' }).click()
  const nav = page.getByRole('complementary', { name: '管理导航' })
  await expect(nav).toBeVisible()
  await nav.getByRole('menuitem', { name: '身份管理' }).click()
  await nav.getByRole('menuitem', { name: '账号管理' }).click()
  await expect(page).toHaveURL(/\/accounts$/)
  await expect(nav).toHaveCount(0)
})

test('tables and dialogs stay inside narrow viewports; action overflow remains usable', async ({ page }, testInfo) => {
  await mockApi(page)
  await installSession(page)
  await page.setViewportSize({ width: 360, height: 800 })
  await page.goto('/accounts')
  await page.getByRole('button', { name: '创建账号' }).click()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  const dialogBounds = await dialog.boundingBox()
  expect(dialogBounds).not.toBeNull()
  expect(dialogBounds!.x).toBeGreaterThanOrEqual(0)
  expect(dialogBounds!.x + dialogBounds!.width).toBeLessThanOrEqual(361)
  const dialogShot = testInfo.outputPath('screenshots', 'account-create-dialog-360x800.png')
  await mkdir(dirname(dialogShot), { recursive: true })
  await page.screenshot({ path: dialogShot, fullPage: true })
  await page.keyboard.press('Escape')
  const row = page.getByRole('row', { name: /Alice Example/ })
  await clickAccountAction(page, /Alice Example/, '详情')
  const drawer = page.getByRole('dialog', { name: '账号详情' })
  await expect(drawer).toBeVisible()
  const drawerShot = testInfo.outputPath('screenshots', 'account-detail-drawer-360x800.png')
  await mkdir(dirname(drawerShot), { recursive: true })
  await page.screenshot({ path: drawerShot, fullPage: true })
  const drawerBounds = await drawer.boundingBox()
  expect(drawerBounds).not.toBeNull()
  expect(drawerBounds!.width).toBeLessThanOrEqual(360)
  const pageWidth = await page.evaluate(() => document.documentElement.scrollWidth)
  expect(pageWidth).toBeLessThanOrEqual(360)
})

test('all eleven routes render without page overflow across six Chromium viewports', async ({ browser }, testInfo) => {
  test.setTimeout(300_000)
  const viewports = [
    { width: 1920, height: 1080 }, { width: 1440, height: 900 }, { width: 1024, height: 768 },
    { width: 768, height: 1024 }, { width: 390, height: 844 }, { width: 360, height: 800 },
  ]
  const routes = [
    { path: '/login', heading: '管理员登录', public: true },
    { path: '/bootstrap', heading: '初始化超级管理员', public: true },
    { path: '/dashboard', heading: '运行概览' },
    { path: '/accounts', heading: '账号管理' },
    { path: '/roles', heading: '角色管理' },
    { path: '/authorization/standard', heading: '权限配置' },
    { path: '/authorization/engine', heading: '权限引擎' },
    { path: '/audit', heading: '操作审计' },
    { path: '/system', heading: '系统状态' },
    { path: '/profile', heading: '个人中心' },
    { path: '/missing-page', heading: 'ROUTE NOT FOUND', public: true },
  ]
  const screenshotRoutes = new Set(['/login', '/bootstrap', '/dashboard', '/accounts', '/roles', '/authorization/engine', '/missing-page'])
  const captures: string[] = []
  for (const viewport of viewports) {
    const context = await browser.newContext({ viewport })
    const page = await context.newPage()
    const consoleErrors: string[] = []
    const pageErrors: string[] = []
    page.on('console', message => { if (message.type() === 'error') consoleErrors.push(message.text()) })
    page.on('pageerror', error => pageErrors.push(error.message))
    await mockApi(page)
    for (const route of routes) {
      await page.addInitScript(isPublic => {
        if (isPublic) sessionStorage.removeItem('awesome-zero-admin.tokens')
        else sessionStorage.setItem('awesome-zero-admin.tokens', JSON.stringify({ accessToken: 'test-access', refreshToken: 'test-refresh', accessExpiresAt: '2027-01-01T00:00:00Z', refreshExpiresAt: '2027-02-01T00:00:00Z' }))
      }, Boolean(route.public))
      await page.goto(route.path)
      await expect(page.getByRole('heading', { name: route.heading, exact: false }).first()).toBeVisible()
      const dimensions = await page.evaluate(() => ({ viewport: window.innerWidth, document: document.documentElement.scrollWidth, body: document.body.scrollWidth }))
      expect(dimensions.document, `${route.path} document width at ${viewport.width}`).toBeLessThanOrEqual(viewport.width + 1)
      expect(dimensions.body, `${route.path} body width at ${viewport.width}`).toBeLessThanOrEqual(viewport.width + 1)
      if (screenshotRoutes.has(route.path) && [1920, 390].includes(viewport.width)) {
        const file = testInfo.outputPath('screenshots', `${route.path.replaceAll('/', '_').replace(/^_/, '') || 'home'}-${viewport.width}x${viewport.height}.png`)
        await mkdir(dirname(file), { recursive: true })
        await page.screenshot({ path: file, fullPage: true })
        captures.push(file)
      }
    }
    expect(consoleErrors, `console errors at ${viewport.width}`).toEqual([])
    expect(pageErrors, `page errors at ${viewport.width}`).toEqual([])
    await context.close()
  }
  expect(captures.length).toBeGreaterThanOrEqual(14)
})
