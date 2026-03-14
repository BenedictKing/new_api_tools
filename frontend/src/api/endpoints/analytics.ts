import type { AuthClient } from '../client'

export function createAnalyticsApi(client: AuthClient) {
  return {
    // GET /analytics/summary — no query params per swagger
    summary: () =>
      client.GET('/analytics/summary', {}),
    state: () =>
      client.GET('/analytics/state', {}),
    syncStatus: () =>
      client.GET('/analytics/sync-status', {}),
    models: (params?: { period?: string; limit?: number }) =>
      client.GET('/analytics/models', { params: { query: params } }),
    usersRequests: (params?: { period?: string; limit?: number }) =>
      client.GET('/analytics/users/requests', { params: { query: params } }),
    usersQuota: (params?: { period?: string; limit?: number }) =>
      client.GET('/analytics/users/quota', { params: { query: params } }),
    process: (body?: Record<string, unknown>) =>
      client.POST('/analytics/process', { body: body as never }),
    batch: (body?: Record<string, unknown>) =>
      client.POST('/analytics/batch', { body: body as never }),
    reset: () =>
      client.POST('/analytics/reset', {}),
    checkConsistency: () =>
      client.POST('/analytics/check-consistency', {}),
  }
}
