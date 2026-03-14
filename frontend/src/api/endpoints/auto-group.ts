import type { AuthClient } from '../client'

export function createAutoGroupApi(client: AuthClient) {
  return {
    config: () =>
      client.GET('/auto-group/config', {}),
    updateConfig: (body: Record<string, unknown>) =>
      client.POST('/auto-group/config', { body: body as never }),
    groups: () =>
      client.GET('/auto-group/groups', {}),
    stats: () =>
      client.GET('/auto-group/stats', {}),
    logs: (params?: { page?: number; page_size?: number }) =>
      client.GET('/auto-group/logs', { params: { query: params } }),
    // GET /auto-group/preview (not POST per swagger)
    preview: (params?: { page?: number; page_size?: number }) =>
      client.GET('/auto-group/preview', { params: { query: params } }),
    scan: (body?: Record<string, unknown>) =>
      client.POST('/auto-group/scan', { body: body as never }),
    batchMove: (body: { user_ids: number[]; target_group: string }) =>
      client.POST('/auto-group/batch-move', { body: body as never }),
    revert: (body: { log_id: number }) =>
      client.POST('/auto-group/revert', { body: body as never }),
  }
}
