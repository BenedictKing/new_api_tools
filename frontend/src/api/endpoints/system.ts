import type { AuthClient } from '../client'

export function createSystemApi(client: AuthClient) {
  return {
    health: () =>
      client.GET('/health', {}),
    healthDb: () =>
      client.GET('/health/db', {}),
    warmupStatus: () =>
      client.GET('/system/warmup-status', {}),
    scale: () =>
      client.GET('/system/scale', {}),
    refreshScale: () =>
      client.POST('/system/scale/refresh', {}),
    indexes: () =>
      client.GET('/system/indexes', {}),
    ensureIndexes: () =>
      client.POST('/system/indexes/ensure', {}),
  }
}
