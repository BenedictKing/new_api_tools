import type { AuthClient } from '../client'

export function createTokensApi(client: AuthClient) {
  return {
    list: (params?: {
      page?: number
      page_size?: number
      status?: string
      name?: string
      group?: string
      user_id?: number
      expired?: 'yes' | 'no'
    }) =>
      client.GET('/tokens', { params: { query: params } }),
    statistics: () =>
      client.GET('/tokens/statistics', {}),
  }
}
