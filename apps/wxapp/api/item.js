const { request } = require('../utils/request')

// 公开浏览(仅上架): {page,page_size,category_id}
exports.list = p => request({ url: '/items', data: p, auth: false })
// 我的物品(需登录,含下架态): owner_id 须为本人 uid
exports.mine = uid => request({ url: '/items', data: { owner_id: uid, page_size: 100 } })
// 搜索(公开): {q, city, category_id, min_price, max_price, order_by, page, page_size, lat, lng, radius_km}
exports.search = p => request({ url: '/items/search', data: p, auth: false })
exports.detail = id => request({ url: '/items/' + id, auth: false })
exports.create = d => request({ url: '/items', method: 'POST', data: d })
exports.update = (id, d) => request({ url: '/items/' + id, method: 'PUT', data: d })
exports.offshelf = id => request({ url: '/items/' + id + '/offshelf', method: 'POST', data: {} })
