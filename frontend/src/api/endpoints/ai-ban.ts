import type { AuthClient } from '../client'

export function createAiBanApi(client: AuthClient) {
  return {
    config: () =>
      client.GET('/ai-ban/config', {}),
    updateConfig: (body: Record<string, unknown>) =>
      client.POST('/ai-ban/config', { body: body as never }),
    // POST /ai-ban/models
    models: (body?: Record<string, unknown>) =>
      client.POST('/ai-ban/models', { body: (body ?? {}) as never }),
    suspiciousUsers: (params?: { window?: string; limit?: number }) =>
      client.GET('/ai-ban/suspicious-users', { params: { query: params } }),
    auditLogs: (params?: { page?: number; page_size?: number; status?: string }) =>
      client.GET('/ai-ban/audit-logs', { params: { query: params } }),
    clearAuditLogs: () =>
      client.DELETE('/ai-ban/audit-logs', {}),
    assess: (body: { user_id: number; window?: string }) =>
      client.POST('/ai-ban/assess', { body: body as never }),
    scan: (body?: Record<string, unknown>) =>
      client.POST('/ai-ban/scan', { body: body as never }),
    whitelist: () =>
      client.GET('/ai-ban/whitelist', {}),
    addWhitelist: (body: { user_id: number }) =>
      client.POST('/ai-ban/whitelist/add', { body: body as never }),
    removeWhitelist: (body: { user_id: number }) =>
      client.POST('/ai-ban/whitelist/remove', { body: body as never }),
    searchWhitelist: (params?: { q?: string }) =>
      client.GET('/ai-ban/whitelist/search', { params: { query: params } }),
    testConnection: () =>
      client.POST('/ai-ban/test-connection', {}),
    testModel: (params: { api_key: string; base_url: string; model: string }) =>
      client.POST('/ai-ban/test-model', { params: { query: params } }),
    resetApiHealth: () =>
      client.POST('/ai-ban/reset-api-health', {}),
  }
}
