import type { AuthClient } from '../client'

export function createStorageApi(client: AuthClient) {
  return {
    info: () =>
      client.GET('/storage/info', {}),
    config: () =>
      client.GET('/storage/config', {}),
    // GET /storage/config/{key} (not POST per swagger)
    getConfigKey: (key: string) =>
      client.GET('/storage/config/{key}', { params: { path: { key } } }),
    deleteConfigKey: (key: string) =>
      client.DELETE('/storage/config/{key}', { params: { path: { key } } }),
    cacheInfo: () =>
      client.GET('/storage/cache/info', {}),
    cacheStats: () =>
      client.GET('/storage/cache/stats', {}),
    // DELETE /storage/cache/dashboard
    deleteCacheDashboard: () =>
      client.DELETE('/storage/cache/dashboard', {}),
    // DELETE /storage/cache
    deleteCache: (body?: { keys?: string[] }) =>
      client.DELETE('/storage/cache', { body: body as never }),
    cacheCleanup: () =>
      client.POST('/storage/cache/cleanup', {}),
    cacheCleanupExpired: () =>
      client.POST('/storage/cache/cleanup-expired', {}),
  }
}
