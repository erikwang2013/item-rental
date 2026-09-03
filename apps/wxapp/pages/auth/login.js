const auth = require('../../api/auth')

Page({
  data: {
    phone: '',
    code: '',
    sending: false,
    countdown: 0
  },
  timer: null,

  onUnload() { if (this.timer) clearInterval(this.timer) },

  onPhone(e) { this.setData({ phone: e.detail.value }) },
  onCode(e) { this.setData({ code: e.detail.value }) },

  sendSms() {
    const phone = this.data.phone.trim()
    if (!/^1\d{10}$/.test(phone)) {
      wx.showToast({ title: '请输入正确的手机号', icon: 'none' })
      return
    }
    if (this.data.sending) return
    this.setData({ sending: true })
    auth.sendSms(phone).then(() => {
      wx.showToast({ title: '验证码已发送(测试码 123456)', icon: 'none' })
      let n = 60
      this.setData({ countdown: n })
      this.timer = setInterval(() => {
        n -= 1
        if (n <= 0) { clearInterval(this.timer); this.timer = null; this.setData({ countdown: 0, sending: false }) }
        else this.setData({ countdown: n })
      }, 1000)
    }).catch(() => {
      this.setData({ sending: false })
    })
  },

  login() {
    const phone = this.data.phone.trim()
    const code = this.data.code.trim()
    if (!/^1\d{10}$/.test(phone)) return wx.showToast({ title: '请输入正确的手机号', icon: 'none' })
    if (!code) return wx.showToast({ title: '请输入验证码', icon: 'none' })
    wx.showLoading({ title: '登录中' })
    auth.login(phone, code).then(d => {
      wx.hideLoading()
      wx.setStorageSync('access_token', d.access_token)
      wx.setStorageSync('refresh_token', d.refresh_token)
      wx.setStorageSync('user', d.user)
      wx.showToast({ title: '登录成功' })
      setTimeout(() => wx.switchTab({ url: '/pages/home/index' }), 600)
    }).catch(() => wx.hideLoading())
  }
})
