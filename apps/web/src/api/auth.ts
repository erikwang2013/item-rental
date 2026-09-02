import { post } from '@/request/index'
export const sms = (phone: string) => post('/auth/sms', { phone }, false)
export const login = (phone: string, code: string) =>
  post<{ access_token:string;refresh_token:string;access_expires_in:number;refresh_expires_in:number }>(
    '/auth/login', { phone, code }, false)
export const refresh = (refresh_token: string) =>
  post('/auth/refresh', { refresh_token }, false)
