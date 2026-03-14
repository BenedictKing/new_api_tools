/**
 * API 层统一导出
 * 使用方式：
 *   const { token } = useAuth()
 *   const api = createDashboardApi(createAuthClient(token))
 *   const { data, error } = await api.overview()
 */

export { createAuthClient, publicClient } from './client'
export type { AuthClient } from './client'

export { createAuthApi, authApi } from './endpoints/auth'
export { createDashboardApi } from './endpoints/dashboard'
export { createTopUpsApi } from './endpoints/topups'
export { createUsersApi } from './endpoints/users'
export { createTokensApi } from './endpoints/tokens'
export { createRedemptionsApi } from './endpoints/redemptions'
export { createRiskApi } from './endpoints/risk'
export { createIpMonitoringApi } from './endpoints/ip-monitoring'
export { createAiBanApi } from './endpoints/ai-ban'
export { createAnalyticsApi } from './endpoints/analytics'
export { createModelStatusApi } from './endpoints/model-status'
export { createSystemApi } from './endpoints/system'
export { createStorageApi } from './endpoints/storage'
export { createAutoGroupApi } from './endpoints/auto-group'

export type { paths, components } from './schema'
