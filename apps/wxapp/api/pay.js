const { request } = require('../utils/request')

// 统一下单(mock 模式返回 prepay:{prepay_id,code_url,mock});channel: native|jsapi
exports.unifiedorder = d => request({ url: '/pay/unifiedorder', method: 'POST', data: d })
