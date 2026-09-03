const messageApi = require('../../api/message')
const util = require('../../utils/util')
const { isLoggedIn } = require('../../utils/request')

Page({
  data: { msgs: [], unread: 0, onlyUnread: false, loading: false },

  onShow() {
    if (!isLoggedIn()) return wx.reLaunch({ url: '/pages/auth/login' })
    this.reload()
  },
  onPullDownRefresh() { this.reload().then(() => wx.stopPullDownRefresh()) },

  reload() {
    this.setData({ msgs: [], loading: false })
    return this.load()
  },

  load() {
    if (this.data.loading) return Promise.resolve()
    this.setData({ loading: true })
    return messageApi.list({ unread: this.data.onlyUnread ? 1 : 0, page: 1, page_size: 50 }).then(d => {
      const msgs = (d.messages || []).map(m => Object.assign({}, m, {
        typeText: util.msgTypeText(m.type),
        timeText: util.fmtTime(m.created_at)
      }))
      this.setData({ msgs, unread: d.unread || 0, loading: false })
    }).catch(() => this.setData({ loading: false }))
  },

  toggleFilter() {
    this.setData({ onlyUnread: !this.data.onlyUnread })
    this.reload()
  },

  // 点击:未读则调 /messages/:id/read 标已读,角标-1
  onTap(e) {
    const id = e.currentTarget.dataset.id
    const msg = this.data.msgs.find(m => m.id === id)
    if (!msg || msg.read) return
    messageApi.markRead(id).then(() => {
      wx.showToast({ title: '已读' })
      this.setData({ msgs: this.data.msgs.map(m => m.id === id ? Object.assign({}, m, { read: true }) : m) })
      this.refreshUnread()
    }).catch(() => {})
  },

  refreshUnread() {
    messageApi.list({ page: 1, page_size: 1 }).then(d => this.setData({ unread: d.unread || 0 })).catch(() => {})
  }
})
