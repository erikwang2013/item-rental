import { post } from '@/request/index'
export const unifiedOrder = (order_id: number) =>
  post<{ timeStamp:string;nonceStr:string;package:string;paySign:string;appId?:string }>(
    '/pay/unifiedorder', { order_id })
