const { request } = require('../utils/request')

exports.create = d => request({ url: '/orders', method: 'POST', data: d }) // {item_id,start_date,end_date}
exports.list = p => request({ url: '/orders', data: p }) // {status?,page,page_size}
exports.detail = id => request({ url: '/orders/' + id })
// 流转:pickup/return_request/return_confirm/breach/cancel
exports.flow = (id, action) => request({ url: '/orders/' + id + '/' + action, method: 'POST', data: {} })
