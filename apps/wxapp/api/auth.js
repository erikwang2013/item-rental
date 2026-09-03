const { request } = require('../utils/request')

exports.sendSms = phone => request({ url: '/auth/sms', method: 'POST', data: { phone }, auth: false })
exports.login = (phone, code) => request({ url: '/auth/login', method: 'POST', data: { phone, code }, auth: false })
// 登出:调后端使 refresh 会话失效(JWT)+ 本地清 token(由调用方清 storage)
exports.logout = () => request({ url: '/auth/logout', method: 'POST', data: {} })
