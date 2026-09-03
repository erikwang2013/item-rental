// wx.request 封装:信封 {code,msg,data} 解析 + Bearer 注入 + 401 双 token 刷新重试(单活跃)
const config = require('../config')

const TOKEN_KEYS = ['access_token', 'refresh_token']
let refreshing = null // 单活跃 refresh Promise

function clearAuth() {
  TOKEN_KEYS.forEach(k => wx.removeStorageSync(k))
  wx.removeStorageSync('user')
}

function redirectLogin() {
  const pages = getCurrentPages()
  const cur = pages.length ? pages[pages.length - 1].route : ''
  if (cur !== 'pages/auth/login') wx.reLaunch({ url: '/pages/auth/login' })
}

// opts: {url, method, data, auth(默认true), toast(默认true,false=静默)}
function request(opts) {
  return new Promise((resolve, reject) => {
    const header = { 'Content-Type': 'application/json' }
    if (opts.auth !== false) {
      const at = wx.getStorageSync('access_token')
      if (at) header.Authorization = 'Bearer ' + at
    }
    wx.request({
      url: config.BASE_URL + opts.url,
      method: opts.method || 'GET',
      data: opts.data || {},
      header,
      timeout: 15000,
      success(res) {
        const body = res.data || {}
        const code = typeof body.code === 'number' ? body.code : res.statusCode
        if (code === 401 || res.statusCode === 401) {
          if (opts.auth === false || opts._retried) {
            // 登录接口自身 401 或重试后仍 401:会话失效
            clearAuth()
            if (opts.toast !== false) wx.showToast({ title: body.msg || '登录已过期', icon: 'none' })
            redirectLogin()
            return reject({ code: 401, msg: body.msg || '登录已过期' })
          }
          return refreshThenRetry(opts).then(resolve, reject)
        }
        if (code === 0) return resolve(body.data)
        const err = { code, msg: body.msg || '请求失败' }
        if (opts.toast !== false) wx.showToast({ title: err.msg, icon: 'none' })
        reject(err)
      },
      fail() {
        const err = { code: -1, msg: '网络异常,请确认后端已启动' }
        if (opts.toast !== false) wx.showToast({ title: err.msg, icon: 'none' })
        reject(err)
      }
    })
  })
}

function refreshThenRetry(opts) {
  const rt = wx.getStorageSync('refresh_token')
  if (!rt) {
    clearAuth()
    return Promise.reject({ code: 401, msg: '请重新登录' })
  }
  if (!refreshing) {
    refreshing = new Promise(resolve => {
      wx.request({
        url: config.BASE_URL + '/auth/refresh',
        method: 'POST',
        header: { 'Content-Type': 'application/json' },
        data: { refresh_token: rt },
        timeout: 15000,
        success(res) {
          const body = res.data || {}
          const d = body.data || {}
          if (body.code === 0 && d.access_token) {
            wx.setStorageSync('access_token', d.access_token)
            wx.setStorageSync('refresh_token', d.refresh_token)
            resolve(true)
          } else resolve(false)
        },
        fail() { resolve(false) }
      })
    }).catch(() => false).finally(() => { refreshing = null })
  }
  return refreshing.then(ok => {
    if (!ok) {
      clearAuth()
      if (opts.toast !== false) wx.showToast({ title: '登录已过期,请重新登录', icon: 'none' })
      redirectLogin()
      throw { code: 401, msg: '登录已过期' }
    }
    opts._retried = true
    return request(opts)
  })
}

module.exports = {
  request,
  clearAuth,
  redirectLogin,
  isLoggedIn: () => !!wx.getStorageSync('access_token')
}
