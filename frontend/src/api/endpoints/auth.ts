import type { AuthClient } from '../client'
import { publicClient } from '../client'

export function createAuthApi(client: AuthClient) {
  return {
    logout: () =>
      client.POST('/auth/logout', {}),
  }
}

export const authApi = {
  login: (body: { username: string; password: string }) =>
    publicClient.POST('/auth/login', { body: body as never }),
}
