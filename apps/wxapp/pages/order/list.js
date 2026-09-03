const orderApi = require('../../api/order')
const itemApi = require('../../api/item')
const util = require('../../utils/util')
const { isLoggedIn } = require('../../utils/request')

const TABS = [
  { label: '全部', status: -1 },
  { label: '待支付', status: 0 },
  { label: '待取货', status: 1 },
  { label: '租赁中', status: 2 },
  { label: '待归还', status: 3 },
  { label: '已归还', status: 4 },
  { label: '已取消', status: 5 },
  { label: '违约', status: 6 }
]

Page({
  data: {
    tabs: TABS,
    active: 0,
    orders: [],
    page: 1,
    hasMore: true,
    loading: false,
    loggedIn: false
  },
  // item_id → 标题 缓存,避免同页重复请求
  titleCache: {},

  onShow() {
    if (!isLoggedIn()) return this.setData({ loggedIn: false, orders: [] })
    this.setData({ loggedIn: true })
    this.reload()
  },
  onPullDownRefresh() { this.reload().then(() => wx.stopPullDownRefresh()) },

  reload() {
    this.setData({ orders: [], page: 1, hasMore: true, loading: false })
    return this.load()
  },

  load() {
    if (this.data.loading || !this.data.hasMore) return Promise.resolve()
    this.setData({ loading: true })
    const tab = TABS[this.data.active]
    return orderApi.list({ status: tab.status, page: this.data.page, page_size: 20 }).then(d => {
      const raw = this.data.orders.concat(d.orders || [])
      return this.enrich(raw).then(orders => {
        this.setData({
          orders, page: this.data.page + 1,
          hasMore: orders.length < (d.total || 0),
          loading: false
        })
      })
    }).catch(() => { this.setData({ loading: false }); return Promise.resolve() })
  },

  // 订单不含物品信息,逐单取公开详情补标题(按 item_id 去重缓存)
  enrich(orders) {
    const need = orders.filter(o => !this.titleCache[o.item_id]).map(o => o.item_id)
    const uniq = Array.from(new Set(need))
    return Promise.all(uniq.map(id =>
      itemApi.detail(id).then(it => { this.titleCache[id] = it.title }).catch(() => { this.titleCache[id] = '物品#' + id })
    )).then(() => orders.map(o => Object.assign({}, o, {
      item_title: this.titleCache[o.item_id],
      stText: util.statusText(o.status),
      rentText: util.money(o.rent_amount)
    })))
  },

  onTab(e) {
    const active = Number(e.currentTarget.dataset.i)
    if (active === this.data.active) return
    this.setData({ active })
    this.reload()
  },
  onReachBottom() { this.load() },

  goLogin() { wx.navigateTo({ url: '/pages/auth/login' }) },
  goDetail(e) { wx.navigateTo({ url: '/pages/order/detail?id=' + e.currentTarget.dataset.id }) },
  fmtMoney: util.fmtMoney
})
