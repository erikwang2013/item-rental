const orderApi = require('../../api/order')
const itemApi = require('../../api/item')
const payApi = require('../../api/pay')
const util = require('../../utils/util')

const CONFIRM = {
  cancel: '取消订单后租金将原路退回,确定取消?',
  pickup: '确认已取到物品?取货后押金将冻结',
  return_request: '确认申请归还?',
  return_confirm: '确认物品已归还?将解冻押金并结算',
  breach: '确认判定违约?将扣押金作为赔偿'
}

Page({
  data: { order: null, item: null, busy: false },

  onLoad(options) {
    this.id = Number(options.id)
    this.load()
  },
  onShow() { if (this.id) this.load() },

  load() {
    orderApi.detail(this.id).then(o => {
      const uid = Number((wx.getStorageSync('user') || {}).id)
      const isRenter = !!uid && uid === o.renter_id
      // 对方信息(服务端富化 owner/renter;按角色取对面)
      const counterpart = uid ? (isRenter ? o.owner : o.renter) : null
      this.setData({ order: o, isRenter, counterpart })
      // 补物品标题(订单接口不含物品冗余)
      itemApi.detail(o.item_id).then(it => {
        const cover = util.splitImages(it.images)[0] || ''
        this.setData({ item: Object.assign({}, it, { cover }) })
      }).catch(() => {})
    }).catch(() => {})
  },

  // 对方头像加载失败 → 回落 👤 占位
  onCpImgErr() {
    if (this.data.counterpart) this.setData({ 'counterpart.avatar': '' })
  },

  // 流转操作:均需 confirm 弹窗
  act(e) {
    const action = e.currentTarget.dataset.action
    const tip = CONFIRM[action]
    const run = () => {
      this.setData({ busy: true })
      orderApi.flow(this.id, action).then(() => {
        wx.showToast({ title: '操作成功' })
        this.setData({ busy: false })
        this.load()
      }).catch(() => this.setData({ busy: false }))
    }
    if (!tip) return run()
    wx.showModal({ title: '确认操作', content: tip, success: res => { if (res.confirm) run() } })
  },

  refund() {
    wx.showModal({
      title: '申请退款',
      content: '确认发起退款?mock 模式将直接成功',
      success: res => {
        if (!res.confirm) return
        this.setData({ busy: true })
        payApi.refund({ order_id: this.id }).then(() => {
          wx.showToast({ title: '退款已发起' })
          this.setData({ busy: false })
          this.load()
        }).catch(() => this.setData({ busy: false }))
      }
    })
  },

  goPay() { wx.navigateTo({ url: '/pages/order/pay?order_id=' + this.id }) },
  goItem() { if (this.data.item) wx.navigateTo({ url: '/pages/items/detail?id=' + this.data.item.id }) },
  fmtMoney: util.fmtMoney
})
