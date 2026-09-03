const itemApi = require('../../api/item')
const categoryApi = require('../../api/category')
const util = require('../../utils/util')

Page({
  data: {
    categories: [],
    items: [],
    page: 1,
    hasMore: true,
    loading: false
  },

  onShow() { this.loadAll() },
  onPullDownRefresh() {
    this.loadAll().then(() => wx.stopPullDownRefresh())
  },

  loadAll() {
    this.setData({ items: [], page: 1, hasMore: true })
    return Promise.all([this.loadCategories(), this.loadItems(true)])
  },

  loadCategories() {
    return categoryApi.list().then(list => {
      this.setData({ categories: list || [] })
    }).catch(() => {})
  },

  // WXML 无法调用方法,预计算每件物品的封面
  decorate(items) {
    return items.map(it => Object.assign({}, it, { cover: (it.images || '').split(',')[0] || '' }))
  },

  loadItems(reset) {
    if (this.data.loading || !this.data.hasMore) return Promise.resolve()
    this.setData({ loading: true })
    return itemApi.list({ page: this.data.page, page_size: 10 }).then(d => {
      const items = this.decorate((reset ? [] : this.data.items).concat(d.items || []))
      this.setData({
        items,
        page: this.data.page + 1,
        hasMore: items.length < (d.total || 0),
        loading: false
      })
    }).catch(() => this.setData({ loading: false }))
  },

  onReachBottom() { this.loadItems(false) },

  goSearch() { wx.navigateTo({ url: '/pages/items/search' }) },
  goCategory(e) {
    const { id, name } = e.currentTarget.dataset
    wx.navigateTo({ url: '/pages/items/list?category_id=' + id + '&title=' + encodeURIComponent(name) })
  },
  goDetail(e) { wx.navigateTo({ url: '/pages/items/detail?id=' + e.currentTarget.dataset.id }) },
  fmtMoney: util.fmtMoney
})
