/**
 * API 客户端工厂
 * 基于 openapi-fetch 提供类型安全的 HTTP 客户端
 */

import createClient from 'openapi-fetch'
import type { paths } from './schema'

const apiUrl = import.meta.env.VITE_API_URL || ''

/**
 * 创建带认证头的 API 客户端
 */
export function createAuthClient(token: string) {
  return createClient<paths>({
    baseUrl: `${apiUrl}/api`,
    headers: { Authorization: `Bearer ${token}` },
  })
}

/**
 * 无需认证的公开 API 客户端
 */
export const publicClient = createClient<paths>({ baseUrl: `${apiUrl}/api` })

export type AuthClient = ReturnType<typeof createAuthClient>
