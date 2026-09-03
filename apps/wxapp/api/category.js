const { request } = require('../utils/request')

exports.list = () => request({ url: '/categories', auth: false })
