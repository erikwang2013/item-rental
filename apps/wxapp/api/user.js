const { request } = require('../utils/request')

exports.profile = () => request({ url: '/user/profile' })
exports.updateProfile = data => request({ url: '/user/profile', method: 'PUT', data })
