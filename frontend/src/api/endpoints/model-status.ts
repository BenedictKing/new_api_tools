import type { AuthClient } from '../client'

export function createModelStatusApi(client: AuthClient) {
  return {
    models: () =>
      client.GET('/model-status/models', {}),
    status: (params?: { window?: string }) =>
      client.GET('/model-status/status', { params: { query: params } }),
    statusBatch: (body: { models: string[]; window?: string }) =>
      client.POST('/model-status/status/batch', { body: body as never }),
    statusByModel: (model_name: string, params?: { window?: string }) =>
      client.GET('/model-status/status/{model_name}', {
        params: { path: { model_name }, query: params },
      }),
    windows: () =>
      client.GET('/model-status/windows', {}),
    configSelected: () =>
      client.GET('/model-status/config/selected', {}),
    updateConfigSelected: (body: { models: string[] }) =>
      client.POST('/model-status/config/selected', { body: body as never }),
    configTheme: () =>
      client.GET('/model-status/config/theme', {}),
    updateConfigTheme: (body: Record<string, unknown>) =>
      client.POST('/model-status/config/theme', { body: body as never }),
    configWindow: () =>
      client.GET('/model-status/config/window', {}),
    updateConfigWindow: (body: Record<string, unknown>) =>
      client.POST('/model-status/config/window', { body: body as never }),
    refreshConfig: () =>
      client.POST('/model-status/config/refresh', { body: {} as never }),
    // Embed（公开）接口
    embedModels: () =>
      client.GET('/model-status/embed/models', {}),
    embedStatus: (params?: { window?: string }) =>
      client.GET('/model-status/embed/status', { params: { query: params } }),
    embedStatusBatch: (body: { models: string[]; window?: string }) =>
      client.POST('/model-status/embed/status/batch', { body: body as never }),
    embedStatusByModel: (model_name: string, params?: { window?: string }) =>
      client.GET('/model-status/embed/status/{model_name}', {
        params: { path: { model_name }, query: params },
      }),
    embedWindows: () =>
      client.GET('/model-status/embed/windows', {}),
    embedConfigSelected: () =>
      client.GET('/model-status/embed/config/selected', {}),
  }
}
