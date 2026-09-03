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

  // 选图上传头像(后端直落库),成功后刷新本地展示
  chooseAvatar() {
    wx.chooseMedia({
      count: 1,
      mediaType: ['image'],
      success: res => {
        const fp = res.tempFiles[0].tempFilePath
        wx.showLoading({ title: '上传中' })
        userApi.uploadAvatar(fp).then(d => {
          this.setData({ avatar: d.avatar || '' })
          return userApi.profile()
        }).then(u => {
          wx.setStorageSync('user', u)
          wx.hideLoading()
          wx.showToast({ title: '头像已更新' })
        }).catch(e => {
          wx.hideLoading()
          wx.showToast({ title: (e && e.msg) || '上传失败', icon: 'none' })
        })
      }
    })
  },

  save() {
    const nickname = this.data.nickname.trim()
    if (nickname.length > 64) return wx.showToast({ title: '昵称过长', icon: 'none' })
    if (this.data.saving) return
    this.setData({ saving: true })
    const data = {}
    if (nickname) data.nickname = nickname
    if (this.data.realName.trim()) data.real_name = this.data.realName.trim()
    userApi.updateProfile(data).then(u => {
      wx.setStorageSync('user', u)
      wx.showToast({ title: '已保存' })
      this.setData({ saving: false })
      setTimeout(() => wx.navigateBack(), 600)
    }).catch(() => this.setData({ saving: false }))
  }
})
