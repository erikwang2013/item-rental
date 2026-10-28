const auth = require('../../api/auth')

Page({
  data: { phone: '' },

  onPhone(e) { this.setData({ phone: e.detail.value }) },

  login() {
    const phone = this.data.phone.trim()
    if (!/^1\d{10}$/.test(phone)) return wx.showToast({ title: '请输入正确的手机号', icon: 'none' })
    wx.showLoading({ title: '登录中' })
    auth.login(phone).then(d => {
      wx.hideLoading()
      wx.setStorageSync('access_token', d.access_token)
      wx.setStorageSync('refresh_token', d.refresh_token)
      wx.setStorageSync('user', d.user)
      wx.showToast({ title: '登录成功' })
      setTimeout(() => wx.switchTab({ url: '/pages/home/index' }), 600)
    }).catch(() => wx.hideLoading())
  }
})
