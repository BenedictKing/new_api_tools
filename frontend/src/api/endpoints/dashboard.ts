import type { AuthClient } from '../client'

export function createDashboardApi(client: AuthClient) {
  return {
    overview: () =>
      client.GET('/dashboard/overview', {}),
    // GET /dashboard/usage — query: period
    usage: (params?: { period?: string }) =>
      client.GET('/dashboard/usage', { params: { query: params } }),
    models: (params?: { period?: string; limit?: number }) =>
      client.GET('/dashboard/models', { params: { query: params } }),
    // GET /dashboard/channels — no query params per swagger
    channels: () =>
      client.GET('/dashboard/channels', {}),
    topUsers: (params?: { period?: string; limit?: number; order_by?: string }) =>
      client.GET('/dashboard/top-users', { params: { query: params } }),
    trendsDaily: (params?: { days?: number }) =>
      client.GET('/dashboard/trends/daily', { params: { query: params } }),
    trendsHourly: (params?: { hours?: number }) =>
      client.GET('/dashboard/trends/hourly', { params: { query: params } }),
    ipDistribution: (params?: { window?: string }) =>
      client.GET('/dashboard/ip-distribution', { params: { query: params } }),
    systemInfo: () =>
      client.GET('/dashboard/system-info', {}),
    refreshEstimate: () =>
      client.GET('/dashboard/refresh-estimate', {}),
    invalidateCache: () =>
      client.POST('/dashboard/cache/invalidate', {}),
  }
}
