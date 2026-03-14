import type { AuthClient } from '../client'

export function createRedemptionsApi(client: AuthClient) {
  return {
    list: (params?: {
      page?: number
      page_size?: number
      status?: string
      name?: string
      start_date?: string
      end_date?: string
    }) =>
      client.GET('/redemptions', { params: { query: params } }),
    statistics: () =>
      client.GET('/redemptions/statistics', {}),
    generate: (body: {
      name: string
      quota: number
      count: number
      expired_time?: number
    }) =>
      client.POST('/redemptions/generate', { body: body as never }),
    // DELETE /redemptions/{id}
    deleteById: (id: number) =>
      client.DELETE('/redemptions/{id}', { params: { path: { id } } }),
    // DELETE /redemptions/batch
    batchDelete: (body: { ids: number[] }) =>
      client.DELETE('/redemptions/batch', { body: body as never }),
  }
}
