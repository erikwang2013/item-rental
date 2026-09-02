// 地理距离工具：Haversine 球面距离（仅标准库 math，无第三方依赖）
package services

import "math"

// earthRadiusKm 地球平均半径（公里）
const earthRadiusKm = 6371.0

// HaversineKm 计算两个经纬度点之间的大圆距离（公里）。
// 入参为十进制度数；对经度环绕（跨 ±180° / 0° 经线）天然正确。
func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadiusKm * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// InRadius 判断点 (lat2,lng2) 是否落在以 (lat1,lng1) 为圆心、radiusKm 为半径的圆内。
// radiusKm <= 0 视为未启用过滤，恒返回 false。
func InRadius(radiusKm, lat1, lng1, lat2, lng2 float64) bool {
	if radiusKm <= 0 {
		return false
	}
	return HaversineKm(lat1, lng1, lat2, lng2) <= radiusKm
}
