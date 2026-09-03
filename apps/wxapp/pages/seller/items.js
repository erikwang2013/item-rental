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
    up: false, // 图片上传中
    pics: [], // 发布表单已选图片 [{path, url}] path=本地临时文件,url=上传后地址
    categories: [],
    form: { title: '', category_id: 0, categoryName: '', daily_price: '', deposit: '', stock: '1', city: '', desc: '' }
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

  // 选图:最多 9 张,选中后逐张上传收集 URL(pics[i].url)
  onChooseImg() {
    if (this.data.up) return
    const remain = 9 - this.data.pics.length
    if (remain <= 0) return wx.showToast({ title: '最多 9 张图片', icon: 'none' })
    wx.chooseMedia({
      count: remain,
      mediaType: ['image'],
      success: res => {
        const paths = (res.tempFiles || []).map(f => f.tempFilePath)
        if (!paths.length) return
        const added = paths.map(p => ({ path: p, url: '' }))
        this.setData({ pics: this.data.pics.concat(added), up: true })
        this.uploadSeq(added)
      }
    })
  },
  uploadSeq(added) {
    if (!added.length) return this.setData({ up: false })
    const cur = added.shift()
    itemApi.uploadImage(cur.path).then(d => {
      const url = (d && d.urls && d.urls[0]) || ''
      if (!url) throw new Error('上传无返回')
      cur.url = url
      this.setData({ pics: this.data.pics.slice() })
      this.uploadSeq(added)
    }).catch(() => {
      // 失败:从列表移除该张并中止后续(已传成功的保留)
      const bad = cur.path
      this.setData({ up: false, pics: this.data.pics.filter(p => p.path !== bad) })
      wx.showToast({ title: '图片上传失败,已移除', icon: 'none' })
    })
  },
  // 点缩略图删除(未传/已传都移除;已传的服务器文件留待清理,dev 可接受)
  delPic(e) {
    const i = e.currentTarget.dataset.i
    const pics = this.data.pics.slice()
    pics.splice(i, 1)
    this.setData({ pics })
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
    if (this.data.up) return wx.showToast({ title: '图片上传中', icon: 'none' })
    const urls = this.data.pics.filter(p => p.url).map(p => p.url)
    this.setData({ saving: true })
    itemApi.create({
      title: f.title.trim(),
      category_id: f.category_id,
      daily_price,
      deposit,
      stock,
      city: f.city.trim(),
      images: urls.length ? JSON.stringify(urls) : '',
      desc: f.desc.trim()
    }).then(() => {
      this.setData({ saving: false, showForm: false, pics: [] })
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
