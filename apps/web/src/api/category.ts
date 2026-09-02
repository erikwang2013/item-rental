import { get } from '@/request/index'
export const listCategories = () => get('/categories', undefined, false)
