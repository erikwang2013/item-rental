const itemApi = require('../../api/item')
const util = require('../../utils/util')

Page({
  data: { item: null, images: [] },

  onLoad(options) {
    this.id = options.id // snowflake id 字符串透传,勿 Number 化
    this.load()
  },

  load() {
    itemApi.detail(this.id).then(it => {
      const images = util.splitImages(it.images)
      wx.setNavigationBarTitle({ title: it.title || '物品详情' })
      this.setData({ item: it, images })
    }).catch(() => {
      setTimeout(() => wx.navigateBack(), 800)
    })
  },

  // 房东头像加载失败 → 回落 👤 占位(置空 avatar 触发 wx:else)
  onOwnerImgErr() {
    if (this.data.item && this.data.item.owner) this.setData({ 'item.owner.avatar': '' })
  },

  // 物品下架/不存在时兜底
  rent() {
    if (!this.data.item) return
    if (this.data.item.status !== 1) return wx.showToast({ title: '该物品已下架', icon: 'none' })
    if (!util.requireLogin()) return
    wx.navigateTo({ url: '/pages/order/confirm?item_id=' + this.id })
  },

  fmtMoney: util.fmtMoney
})
