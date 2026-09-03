const { request } = require('../utils/request')

exports.sendSms = phone => request({ url: '/auth/sms', method: 'POST', data: { phone }, auth: false })
exports.login = (phone, code) => request({ url: '/auth/login', method: 'POST', data: { phone, code }, auth: false })
// 登出:调后端使 refresh 会话失效(JWT)+ 本地清 token(由调用方清 storage)
// 带 refresh_token 使后端只撤销本端会话(多端同登时不清他端)
exports.logout = () => request({
  url: '/auth/logout',
  method: 'POST',
  data: { refresh_token: wx.getStorageSync('refresh_token') || '' }
})
