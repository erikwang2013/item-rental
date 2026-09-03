const auth = require('../../api/auth')
const messageApi = require('../../api/message')
const { isLoggedIn, clearAuth } = require('../../utils/request')

Page({
  data: { user: null, unread: 0 },

  onShow() {
    if (!isLoggedIn()) return this.setData({ user: null, unread: 0 })
    const user = wx.getStorageSync('user')
    this.setData({ user })
    this.loadUnread()
    // 若本地资料缺失,从后端拉一次
    if (!user || !user.nickname) this.refreshProfile()
  },

  refreshProfile() {
    require('../../api/user').profile().then(u => {
      wx.setStorageSync('user', u)
      this.setData({ user: u })
    }).catch(() => {})
  },

  // 消息列表响应自带 unread 汇总;page_size=1 足够
  loadUnread() {
    messageApi.list({ page: 1, page_size: 1 }).then(d => {
      this.setData({ unread: d.unread || 0 })
    }).catch(() => {})
  },

  goLogin() { wx.navigateTo({ url: '/pages/auth/login' }) },
  goProfile() { wx.navigateTo({ url: '/pages/user/profile' }) },
  goSeller() { wx.navigateTo({ url: '/pages/seller/items' }) },
  goMessages() { wx.navigateTo({ url: '/pages/messages/index' }) },

  logout() {
    wx.showModal({
      title: '退出登录',
      content: '确定退出当前账号?',
      success: res => {
        if (!res.confirm) return
        // 登出必须调后端使 refresh 会话失效(JWT),再本地清 token
        auth.logout().catch(() => {}).then(() => {
          clearAuth()
          this.setData({ user: null, unread: 0 })
          wx.showToast({ title: '已退出登录' })
        })
      }
    })
  },

})
