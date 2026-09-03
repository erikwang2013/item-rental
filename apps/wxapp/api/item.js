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
const config = require('../config')
const { clearAuth, redirectLogin } = require('../utils/request')

// 物品图上传(多图端点,单张调用,返回 {urls:[...]});wx.uploadFile 不走 request 的 json 路径
exports.uploadImage = filePath => new Promise((resolve, reject) => {
  const at = wx.getStorageSync('access_token')
  wx.uploadFile({
    url: config.BASE_URL + '/items/upload',
    filePath,
    name: 'files',
    header: { Authorization: 'Bearer ' + at },
    success(res) {
      if (res.statusCode === 401) {
        clearAuth()
        redirectLogin()
        return reject({ code: 401, msg: '登录已过期' })
      }
      const body = JSON.parse(res.data || '{}')
      if (body.code === 0) return resolve(body.data)
      reject({ code: body.code || res.statusCode, msg: body.msg || '上传失败' })
    },
    fail() { reject({ code: -1, msg: '网络异常' }) }
  })
})
