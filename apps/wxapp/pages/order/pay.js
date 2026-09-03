const orderApi = require('../../api/order')
const payApi = require('../../api/pay')
const util = require('../../utils/util')

Page({
  data: { order: null, statusText: '', paying: false, polled: false },

  onLoad(options) {
    this.orderId = options.order_id // snowflake id 字符串透传,勿 Number 化
    this.orderNo = options.order_no || ''
    this.load()
  },
  onUnload() { this.stopPoll() },

  load() {
    orderApi.detail(this.orderId).then(o => {
      this.orderNo = o.order_no
      this.setData({ order: o, statusText: util.statusText(o.status) })
    }).catch(() => {})
  },

  // mock 轮询:统一下单后轮询订单状态;真实环境在此替换为 wx.requestPayment(pay_params)
  pay() {
    if (this.data.paying) return
    if (!this.orderNo) return wx.showToast({ title: '缺少订单号', icon: 'none' })
    this.setData({ paying: true })
    payApi.unifiedorder({ order_no: this.orderNo, channel: 'native' }).then(res => {
      wx.showToast({ title: '支付已发起(mock)', icon: 'none' })
      this.startPoll()
    }).catch(() => this.setData({ paying: false }))
  },

  startPoll() {
    let n = 0
    this.timer = setInterval(() => {
      n += 1
      this.load()
      const st = this.data.order ? this.data.order.status : 0
      if (st !== 0 || n >= 10) {
        this.stopPoll()
        this.setData({ polled: true, paying: false })
        // mock 下若服务端未模拟回调,订单停留待支付,提示刷新即可
        if (st === 0) wx.showToast({ title: '未收到支付结果,请稍后刷新订单', icon: 'none' })
      }
    }, 1500)
  },
  stopPoll() { if (this.timer) { clearInterval(this.timer); this.timer = null } },

  goDetail() { wx.redirectTo({ url: '/pages/order/detail?id=' + this.orderId }) },
  goList() { wx.switchTab({ url: '/pages/order/list' }) },
  fmtMoney: util.fmtMoney
})
