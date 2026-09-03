const { request } = require('../utils/request')

// 公开浏览(仅上架): {page,page_size,category_id}
exports.list = p => request({ url: '/items', data: p, auth: false })
// 搜索(公开): {q, city, category_id, min_price, max_price, order_by, page, page_size, lat, lng, radius_km}
exports.search = p => request({ url: '/items/search', data: p, auth: false })
exports.detail = id => request({ url: '/items/' + id, auth: false })
exports.create = d => request({ url: '/items', method: 'POST', data: d })
exports.update = (id, d) => request({ url: '/items/' + id, method: 'PUT', data: d })
exports.offshelf = id => request({ url: '/items/' + id + '/offshelf', method: 'POST', data: {} })
