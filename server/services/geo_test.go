// 地理距离工具单元测试：纯 math 计算，无 DB/网络依赖，可离线运行。
package services

import (
	"math"
	"testing"
)

func TestHaversineKmKnownDistance(t *testing.T) {
	// 北京天安门 (39.9042, 116.4074) → 上海人民广场 (31.2304, 121.4737)
	got := HaversineKm(39.9042, 116.4074, 31.2304, 121.4737)
	want := 1067.0 // 公里，实际约 1067 km
	if math.Abs(got-want) > 15 {
		t.Errorf("北京-上海距离 = %.2f km, 期望 ≈ %.0f km", got, want)
	}
}

func TestHaversineKmZeroDistance(t *testing.T) {
	if got := HaversineKm(30.0, 120.0, 30.0, 120.0); got != 0 {
		t.Errorf("同点距离 = %v, 期望 0", got)
	}
	if got := HaversineKm(0, 0, 0, 0); got != 0 {
		t.Errorf("原点距离 = %v, 期望 0", got)
	}
}

func TestHaversineKmCrossPrimeMeridian(t *testing.T) {
	// 跨 0° 经线：赤道上 (0,1) → (0,-1)，弧长 = 2° ≈ 222.4 km
	got := HaversineKm(0, 1, 0, -1)
	want := 2.0 * math.Pi / 180 * 6371.0 // 222.39
	if math.Abs(got-want) > 0.5 {
		t.Errorf("跨0°经线距离 = %.2f km, 期望 ≈ %.2f km", got, want)
	}
}

func TestInRadius(t *testing.T) {
	centerLat, centerLng := 39.9042, 116.4074 // 北京
	// 上海距北京 ~1067km：1000km 圈外，1100km 圈内
	if InRadius(1000, centerLat, centerLng, 31.2304, 121.4737) {
		t.Error("1000km 半径不应包含上海")
	}
	if !InRadius(1100, centerLat, centerLng, 31.2304, 121.4737) {
		t.Error("1100km 半径应包含上海")
	}
	// 同点零距离在任意正半径内
	if !InRadius(0.5, centerLat, centerLng, centerLat, centerLng) {
		t.Error("同点应在半径内")
	}
	// radiusKm <= 0：过滤未启用，恒 false
	if InRadius(0, centerLat, centerLng, centerLat, centerLng) {
		t.Error("radiusKm=0 不应命中")
	}
	if InRadius(-5, centerLat, centerLng, centerLat, centerLng) {
		t.Error("radiusKm<0 不应命中")
	}
}

func TestInRadiusMissingCoordinates(t *testing.T) {
	// 物品坐标缺失（lat/lng 为 0）时安全计算，不 panic、不误判为"圈内零距离"
	if InRadius(50, 30, 120, 0, 0) {
		t.Error("缺失坐标的物品不应被判为圈内")
	}
	// 查询中心缺失同理安全
	if InRadius(50, 0, 0, 30, 120) {
		t.Error("缺失查询中心不应判命中")
	}
	// 确保中间计算没有产生 NaN
	if d := HaversineKm(0, 0, 30, 120); math.IsNaN(d) {
		t.Errorf("缺失坐标 Haversine 不应产生 NaN, got %v", d)
	}
	// 缺失坐标是纯数学层不关心的"业务约定"：距离本身仍按真实球面距离计算，
	// 全零坐标（等点）距离为 0，非 NaN —— 具体跳过策略由调用方（SearchItems）决定。
	if d := HaversineKm(0, 0, 0, 0); d != 0 || math.IsNaN(d) {
		t.Errorf("全零坐标距离 = %v, 期望 0 且非 NaN", d)
	}
}
