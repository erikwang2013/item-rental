const { request } = require('../utils/request')

// 统一下单(mock 模式返回 prepay:{prepay_id,code_url,mock});channel: native|jsapi
exports.unifiedorder = d => request({ url: '/pay/unifiedorder', method: 'POST', data: d })
// 退款 {order_id, refund_reason}(mock 直接成功)
exports.refund = d => request({ url: '/pay/refund', method: 'POST', data: d })
