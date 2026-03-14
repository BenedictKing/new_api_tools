import type { AuthClient } from '../client'

export function createUsersApi(client: AuthClient) {
  return {
    list: (params?: {
      page?: number
      page_size?: number
      search?: string
      status?: number
      order_by?: string
    }) =>
      client.GET('/users', { params: { query: params } }),
    stats: (params?: { quick?: boolean }) =>
      client.GET('/users/stats', { params: { query: params } }),
    banned: (params?: { page?: number; page_size?: number }) =>
      client.GET('/users/banned', { params: { query: params } }),
    softDeletedCount: () =>
      client.GET('/users/soft-deleted/count', {}),
    purgeSoftDeleted: (body?: { dry_run?: boolean }) =>
      client.POST('/users/soft-deleted/purge', { body: (body ?? {}) as never }),
    // DELETE /users/{user_id}
    deleteById: (user_id: number) =>
      client.DELETE('/users/{user_id}', { params: { path: { user_id } } }),
    ban: (user_id: number, body?: Record<string, unknown>) =>
      client.POST('/users/{user_id}/ban', { params: { path: { user_id } }, body: body as never }),
    unban: (user_id: number) =>
      client.POST('/users/{user_id}/unban', { params: { path: { user_id } } }),
    invited: (user_id: number, params?: { page?: number; page_size?: number }) =>
      client.GET('/users/{user_id}/invited', { params: { path: { user_id }, query: params } }),
    batchDelete: (body: { user_ids: number[] }) =>
      client.POST('/users/batch-delete', { body: body as never }),
    disableToken: (token_id: number) =>
      client.POST('/users/tokens/{token_id}/disable', { params: { path: { token_id } } }),
  }
}
