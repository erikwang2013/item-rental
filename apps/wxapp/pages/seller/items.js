const itemApi = require('../../api/item')
const categoryApi = require('../../api/category')
const util = require('../../utils/util')
const { isLoggedIn } = require('../../utils/request')

// ponytail: 后端无"我发布的物品"列表接口(GET /items 仅公开上架列表,无 owner 过滤),
// 本页记录本设备发布过的 item_id 于本地存储,再逐条拉公开详情展示(含下架态)。
const MY_IDS_KEY = 'my_item_ids'

Page({
  data: {
    loggedIn: false,
    items: [],
    showForm: false,
    saving: false,
    categories: [],
    form: { title: '', category_id: 0, categoryName: '', daily_price: '', deposit: '', stock: '1', city: '', images: '', desc: '' }
  },

  onShow() {
    if (!isLoggedIn()) return this.setData({ loggedIn: false })
    this.setData({ loggedIn: true })
    this.loadCategories()
    this.reload()
  },
  onPullDownRefresh() { this.reload().then(() => wx.stopPullDownRefresh()) },

  myIds() { return wx.getStorageSync(MY_IDS_KEY) || [] },

  reload() {
    const ids = this.myIds()
    if (!ids.length) { this.setData({ items: [] }); return Promise.resolve() }
    // 逐条取公开详情;404/删除的静默丢弃
    return Promise.all(ids.map(id =>
      itemApi.detail(id).then(it =>
        Object.assign({}, it, { cover: util.splitImages(it.images)[0] || '', stText: it.status === 1 ? '上架中' : '已下架' })
      ).catch(() => null)
    )).then(list => {
      const items = list.filter(Boolean)
      this.setData({ items })
      const alive = items.map(i => i.id)
      wx.setStorageSync(MY_IDS_KEY, alive)
    })
  },

  loadCategories() {
    if (this.data.categories.length) return
    categoryApi.list().then(list => this.setData({ categories: list || [] })).catch(() => {})
  },

  toggleForm() { this.setData({ showForm: !this.data.showForm }) },
  onField(e) {
    const key = e.currentTarget.dataset.k
    this.setData({ ['form.' + key]: e.detail.value })
  },
  onCategory(e) {
    const i = Number(e.detail.value)
    const c = this.data.categories[i]
    if (c) this.setData({ 'form.category_id': c.id, 'form.categoryName': c.name })
  },

  publish() {
    const f = this.data.form
    const daily_price = Number(f.daily_price)
    const deposit = Number(f.deposit)
    const stock = parseInt(f.stock, 10)
    if (!f.title.trim()) return wx.showToast({ title: '请填写标题', icon: 'none' })
    if (!f.category_id) return wx.showToast({ title: '请选择类目', icon: 'none' })
    if (!daily_price || daily_price <= 0) return wx.showToast({ title: '请填写日租金', icon: 'none' })
    if (deposit < 0) return wx.showToast({ title: '押金不合法', icon: 'none' })
    if (!stock || stock < 1 || stock > 999) return wx.showToast({ title: '库存需在 1-999', icon: 'none' })
    this.setData({ saving: true })
    itemApi.create({
      title: f.title.trim(),
      category_id: f.category_id,
      daily_price,
      deposit,
      stock,
      city: f.city.trim(),
      images: f.images.trim(),
      desc: f.desc.trim()
    }).then(it => {
      const ids = this.myIds()
      if (ids.indexOf(it.id) < 0) wx.setStorageSync(MY_IDS_KEY, [it.id].concat(ids))
      this.setData({ saving: false, showForm: false })
      wx.showToast({ title: '发布成功' })
      this.reload()
    }).catch(() => this.setData({ saving: false }))
  },

  // 下架(仅 owner,后端校验)
  offshelf(e) {
    const id = e.currentTarget.dataset.id
    wx.showModal({
      title: '下架物品',
      content: '下架后不再被搜索与租用,确定?',
      success: res => {
        if (!res.confirm) return
        itemApi.offshelf(id).then(() => {
          wx.showToast({ title: '已下架' })
          this.reload()
        }).catch(() => {})
      }
    })
  },

  goLogin() { wx.navigateTo({ url: '/pages/auth/login' }) },
  fmtMoney: util.fmtMoney
})
