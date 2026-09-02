type Envelope<T = any> = { code: number; msg: string; data: T }

export interface Opts {
  url: string
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
  data?: any
  headers?: Record<string, string>
  auth?: boolean
}

export class ApiError extends Error {
  constructor(public code: number, msg: string, public data?: any) {
    super(msg || `api error ${code}`)
    this.name = 'ApiError'
  }
}

const BASE = (import.meta as any).env?.VITE_API_BASE_URL || '/api/v1'

async function token(): Promise<string | undefined> {
  try {
    const { useUserStore } = await import('@/stores/user')
    return useUserStore().accessToken
  } catch { return undefined }
}

export async function request<T>(opts: Opts): Promise<T> {
  const auth = opts.auth !== false
  const tk = auth ? await token() : undefined
  return new Promise((resolve, reject) => {
    uni.request({
      url: BASE + opts.url,
      method: opts.method || 'GET',
      data: opts.data,
      headers: {
        'Content-Type': 'application/json',
        ...(tk ? { Authorization: 'Bearer ' + tk } : {}),
        ...opts.headers,
      },
      success: async (res: any) => {
        if (res.statusCode === 401 && auth && tk) {
          try {
            const ok = await (await import('./auth')).refreshAccessToken()
            if (ok) { const r = await request<T>(opts); return resolve(r) }
          } catch {}
          return reject(new ApiError(401, '登录已过期'))
        }
        if (res.statusCode === 403) return reject(new ApiError(403, '无权限'))
        const env = res.data as Envelope<T>
        if (!env || env.code !== 0) return reject(new ApiError(env?.code ?? 500, env?.msg, env?.data))
        resolve(env.data)
      },
      fail: (e: any) => reject(new ApiError(-1, '网络请求失败: ' + e.errMsg)),
    })
  })
}

export function get<T>(url: string, params?: Record<string, any>, auth = true) {
  const qs = params ? '?' + new URLSearchParams(params as any).toString() : ''
  return request<T>({ url: url + qs, method: 'GET', auth })
}
export function post<T>(url: string, data?: any, auth = true) {
  return request<T>({ url, method: 'POST', data, auth })
}
export function put<T>(url: string, data?: any, auth = true) {
  return request<T>({ url, method: 'PUT', data, auth })
}
