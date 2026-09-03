const { request } = require('../utils/request')

// 列表:{messages,total,page,unread};  {unread?,page,page_size}
exports.list = p => request({ url: '/messages', data: p })
exports.markRead = id => request({ url: '/messages/' + id + '/read', method: 'POST', data: {} })
