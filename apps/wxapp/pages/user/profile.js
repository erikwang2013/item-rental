const userApi = require('../../api/user')

Page({
  data: { nickname: '', realName: '', avatar: '', saving: false },

  onLoad() { this.load() },

  load() {
    userApi.profile().then(u => {
      this.setData({ nickname: u.nickname || '', realName: u.real_name || '', avatar: u.avatar || '' })
    }).catch(() => {})
  },

  onNick(e) { this.setData({ nickname: e.detail.value }) },
  onReal(e) { this.setData({ realName: e.detail.value }) },
  // 服务端无图片上传接口,avatar 仅展示(URL 需 http 域名);后端字段支持传入 URL
  onAvatarUrl(e) { this.setData({ avatar: e.detail.value }) },

  save() {
    const nickname = this.data.nickname.trim()
    if (nickname.length > 64) return wx.showToast({ title: '昵称过长', icon: 'none' })
    if (this.data.saving) return
    this.setData({ saving: true })
    const data = {}
    if (nickname) data.nickname = nickname
    if (this.data.realName.trim()) data.real_name = this.data.realName.trim()
    if (this.data.avatar.trim()) data.avatar = this.data.avatar.trim()
    userApi.updateProfile(data).then(u => {
      wx.setStorageSync('user', u)
      wx.showToast({ title: '已保存' })
      this.setData({ saving: false })
      setTimeout(() => wx.navigateBack(), 600)
    }).catch(() => this.setData({ saving: false }))
  }
})
