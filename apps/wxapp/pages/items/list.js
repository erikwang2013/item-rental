const itemApi = require('../../api/item')
const util = require('../../utils/util')

Page({
  data: { items: [], page: 1, hasMore: true, loading: false, title: '' },

  onLoad(options) {
    this.categoryId = Number(options.category_id) || 0
    if (options.title) {
      this.setData({ title: decodeURIComponent(options.title) })
      wx.setNavigationBarTitle({ title: decodeURIComponent(options.title) })
    }
    this.load(true)
  },

  decorate(items) {
    return items.map(it => Object.assign({}, it, { cover: (it.images || '').split(',')[0] || '' }))
  },

  load(reset) {
    if (this.data.loading || !this.data.hasMore) return
    this.setData({ loading: true })
    itemApi.list({ page: this.data.page, page_size: 20, category_id: this.categoryId }).then(d => {
      const items = this.decorate((reset ? [] : this.data.items).concat(d.items || []))
      this.setData({ items, page: this.data.page + 1, hasMore: items.length < (d.total || 0), loading: false })
    }).catch(() => this.setData({ loading: false }))
  },

  onReachBottom() { this.load(false) },
  goDetail(e) { wx.navigateTo({ url: '/pages/items/detail?id=' + e.currentTarget.dataset.id }) },
  fmtMoney: util.fmtMoney
})
