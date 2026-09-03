const config = require('../config')
const { request, clearAuth, redirectLogin } = require('../utils/request')

exports.profile = () => request({ url: '/user/profile' })
exports.updateProfile = data => request({ url: '/user/profile', method: 'PUT', data })
// 头像上传(后端直落库,返回 {avatar: URL});wx.uploadFile 不走 request 的 json 路径,自包含
exports.uploadAvatar = filePath => new Promise((resolve, reject) => {
  const at = wx.getStorageSync('access_token')
  wx.uploadFile({
    url: config.BASE_URL + '/user/avatar',
    filePath,
    name: 'file',
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
