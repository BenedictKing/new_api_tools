import type { AuthClient } from '../client'

export function createIpMonitoringApi(client: AuthClient) {
  return {
    stats: () =>
      client.GET('/ip-monitoring/stats', {}),
    multiIpTokens: (params?: { page?: number; page_size?: number; min_ips?: number }) =>
      client.GET('/ip-monitoring/multi-ip-tokens', { params: { query: params } }),
    multiIpUsers: (params?: { page?: number; page_size?: number; min_ips?: number }) =>
      client.GET('/ip-monitoring/multi-ip-users', { params: { query: params } }),
    // swagger: query params are min_tokens, limit, window
    sharedIps: (params?: { min_tokens?: number; limit?: number; window?: string }) =>
      client.GET('/ip-monitoring/shared-ips', { params: { query: params } }),
    // swagger: query params are limit, window
    userIps: (user_id: number, params?: { limit?: number; window?: string }) =>
      client.GET('/ip-monitoring/users/{user_id}/ips', {
        params: { path: { user_id }, query: params },
      }),
    geoBatch: (body: { ips: string[] }) =>
      client.POST('/ip-monitoring/geo/batch', { body: body as never }),
    geoLookup: (ip: string) =>
      client.GET('/ip-monitoring/geo/{ip}', { params: { path: { ip } } }),
    indexStatus: () =>
      client.GET('/ip-monitoring/index-status', {}),
    ensureIndexes: () =>
      client.POST('/ip-monitoring/ensure-indexes', {}),
  }
}
