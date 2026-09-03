const itemApi = require('../../api/item')
const orderApi = require('../../api/order')
const util = require('../../utils/util')

Page({
  data: {
    item: null,
    startDate: '',
    endDate: '',
    days: 1,
    rent: '0'
  },

  onLoad(options) {
    // snowflake id 超出 JS Number 安全整数,全程按字符串透传
    this.itemId = options.item_id
    if (!util.requireLogin()) return
    itemApi.detail(this.itemId).then(it => {
      if (it.status !== 1) {
        wx.showToast({ title: '该物品已下架', icon: 'none' })
        return setTimeout(() => wx.navigateBack(), 800)
      }
      const startDate = util.today()
      const endDate = util.addDays(startDate, 1)
      const cover = util.splitImages(it.images)[0] || ''
      this.setData({ item: Object.assign({}, it, { cover }), startDate, endDate })
      this.calc()
    }).catch(() => setTimeout(() => wx.navigateBack(), 800))
  },

  onStart(e) {
    const startDate = e.detail.value
    // 结束日期不得早于开始次日
    const endDate = util.dayDiff(startDate, this.data.endDate) < 1 ? util.addDays(startDate, 1) : this.data.endDate
    this.setData({ startDate, endDate })
    this.calc()
  },
  onEnd(e) {
    this.setData({ endDate: e.detail.value })
    this.calc()
  },

  calc() {
    const { item, startDate, endDate } = this.data
    const days = util.dayDiff(startDate, endDate)
    const rent = days > 0 ? (item.daily_price * days).toFixed(2) : '0'
    this.setData({ days: days > 0 ? days : 0, rent })
  },

  submit() {
    if (this.data.days < 1) return wx.showToast({ title: '租期不合法', icon: 'none' })
    wx.showLoading({ title: '提交中' })
    orderApi.create({
      item_id: this.itemId,
      start_date: this.data.startDate,
      end_date: this.data.endDate
    }).then(o => {
      wx.hideLoading()
      wx.redirectTo({ url: '/pages/order/pay?order_id=' + o.id + '&order_no=' + o.order_no })
    }).catch(() => wx.hideLoading())
  },

  fmtMoney: util.fmtMoney
})
