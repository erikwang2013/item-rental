const itemApi = require('../../api/item')
const util = require('../../utils/util')

Page({
  data: {
    keyword: '',
    city: '',
    items: [],
    page: 1,
    hasMore: true,
    loading: false,
    searched: false
  },

  // 契约:搜索接口路径为 /items/search,关键词参数名 q(非 keyword),city 为可选过滤
  doSearch() {
    this.setData({ items: [], page: 1, hasMore: true, loading: false, searched: true })
    this.load()
  },

  decorate(items) {
    return items.map(it => Object.assign({}, it, { cover: (it.images || '').split(',')[0] || '' }))
  },

  load() {
    if (this.data.loading || !this.data.hasMore) return
    this.setData({ loading: true })
    const p = { q: this.data.keyword.trim(), city: this.data.city.trim(), page: this.data.page, page_size: 20 }
    itemApi.search(p).then(d => {
      const items = this.decorate(this.data.items.concat(d.items || []))
      this.setData({ items, page: this.data.page + 1, hasMore: items.length < (d.total || 0), loading: false })
    }).catch(() => this.setData({ loading: false }))
  },

  onReachBottom() { this.load() },
  onKw(e) { this.setData({ keyword: e.detail.value }) },
  onCity(e) { this.setData({ city: e.detail.value }) },
  clearCity() { this.setData({ city: '' }) },
  goDetail(e) { wx.navigateTo({ url: '/pages/items/detail?id=' + e.currentTarget.dataset.id }) },
  fmtMoney: util.fmtMoney
})
