import { get, put } from '@/request/index'
export const getProfile = () => get('/user/profile')
export const updateProfile = (d: { nickname?:string;avatar?:string }) => put('/user/profile', d)
