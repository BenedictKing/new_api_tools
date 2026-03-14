import type { AuthClient } from '../client'

export function createTopUpsApi(client: AuthClient) {
  return {
    list: (params?: {
      page?: number
      page_size?: number
      status?: string
      payment_method?: string
      trade_no?: string
      start_date?: string
      end_date?: string
    }) =>
      client.GET('/top-ups', { params: { query: params } }),
    statistics: (params?: { start_date?: string; end_date?: string }) =>
      client.GET('/top-ups/statistics', { params: { query: params } }),
    paymentMethods: () =>
      client.GET('/top-ups/payment-methods', {}),
    getById: (id: number) =>
      client.GET('/top-ups/{id}', { params: { path: { id } } }),
    refund: (id: number) =>
      client.POST('/top-ups/{id}/refund', { params: { path: { id } } }),
  }
}
