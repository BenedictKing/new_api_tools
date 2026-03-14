import type { AuthClient } from '../client'

export function createRiskApi(client: AuthClient) {
  return {
    userAnalysis: (user_id: number, params?: { window?: string }) =>
      client.GET('/risk/users/{user_id}/analysis', {
        params: { path: { user_id }, query: params },
      }),
    leaderboards: (params?: { type?: string; limit?: number; window?: string }) =>
      client.GET('/risk/leaderboards', { params: { query: params } }),
    banRecords: (params?: { page?: number; page_size?: number }) =>
      client.GET('/risk/ban-records', { params: { query: params } }),
    // swagger: query params are min_invited, include_activity, limit
    affiliatedAccounts: (params?: { min_invited?: number; include_activity?: boolean; limit?: number }) =>
      client.GET('/risk/affiliated-accounts', { params: { query: params } }),
    sameIpRegistrations: (params?: { limit?: number }) =>
      client.GET('/risk/same-ip-registrations', { params: { query: params } }),
    tokenRotation: (params?: { limit?: number; window?: string }) =>
      client.GET('/risk/token-rotation', { params: { query: params } }),
  }
}
