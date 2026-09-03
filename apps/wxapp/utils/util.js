// 通用工具:金额/日期/状态文案/图片列表
const ORDER_STATUS = ['待支付', '待取货', '租赁中', '待归还', '已归还', '已取消', '违约']
const MSG_TYPE = {
  payment_success: '支付成功',
  payment_refunded: '退款成功',
  return_confirmed: '归还确认',
  breach: '违约判定',
  order_cancelled: '订单取消'
}

const pad = n => (n < 10 ? '0' + n : '' + n)
const fmtDate = d => d.getFullYear() + '-' + pad(d.getMonth() + 1) + '-' + pad(d.getDate())
const today = () => fmtDate(new Date())
// 日期串加/减 n 天;解析用 / 分隔,规避 iOS 对 - 分隔符的兼容问题
function addDays(dateStr, n) {
  const d = new Date(dateStr.replace(/-/g, '/'))
  d.setDate(d.getDate() + n)
  return fmtDate(d)
}
function dayDiff(a, b) {
  return Math.round((new Date(b.replace(/-/g, '/')) - new Date(a.replace(/-/g, '/'))) / 86400000)
}
// 金额:后端为元(float),去尾零 ¥30 / ¥12.5
function money(v) {
  const n = Number(v)
  if (!isFinite(n)) return '0'
  const s = n.toFixed(2)
  return s.endsWith('.00') ? String(n) : s.replace(/0$/, '')
}
function fmtMoney(v) { return '¥' + money(v) }
// images 逗号分隔串 → 数组
function splitImages(s) {
  return (s || '').split(',').map(x => x.trim()).filter(Boolean)
}
// "2026-09-04T00:14:33+08:00" → "09-04 00:14"
function fmtTime(t) {
  if (!t) return ''
  const s = t.slice(5, 16).replace('T', ' ')
  return s
}
function statusText(st) { return ORDER_STATUS[st] || '未知' }
function msgTypeText(type) { return MSG_TYPE[type] || type || '通知' }

// 登录守卫:未登录则跳登录页并返回 false
function requireLogin() {
  if (wx.getStorageSync('access_token')) return true
  wx.navigateTo({ url: '/pages/auth/login' })
  return false
}

module.exports = {
  ORDER_STATUS, MSG_TYPE, pad, fmtDate, today, addDays, dayDiff,
  money, fmtMoney, splitImages, fmtTime, statusText, msgTypeText, requireLogin
}
