const itemApi = require('../../api/item')
const userApi = require('../../api/user')
const categoryApi = require('../../api/category')
const util = require('../../utils/util')
const { isLoggedIn } = require('../../utils/request')

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

  reload() {
    // 我的物品:先取本人 uid,再拉 owner 视图(含下架态)
    return userApi.profile().then(u => itemApi.mine(u.id)).then(list => {
      const items = (list || []).map(it =>
        Object.assign({}, it, { cover: util.splitImages(it.images)[0] || '', stText: it.status === 1 ? '上架中' : '已下架' })
      )
      this.setData({ items })
    }).catch(() => {})
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
    }).then(() => {
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
