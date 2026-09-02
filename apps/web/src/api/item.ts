import { get, post, put } from '@/request/index'
export const listItems = (p?: any) => get('/items', p, false)
export const searchItems = (p?: any) => get('/items/search', p, false)
export const getItem = (id: number) => get(`/items/${id}`, undefined, false)
export const createItem = (d: any) => post('/items', d)
export const updateItem = (id: number, d: any) => put(`/items/${id}`, d)
export const offshelfItem = (id: number) => post(`/items/${id}/offshelf`)
