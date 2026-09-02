// 物品输入校验单元测试：纯函数，无 DB 依赖，可离线运行
package services

import (
	"strings"
	"testing"
)

func f(n float64) *float64 { return &n }

func validItemReq() ItemValidateRequest {
	return ItemValidateRequest{
		Title:      "相机",
		Images:     `["https://example.com/a.jpg"]`,
		DailyPrice: 50,
		Deposit:    100,
		Stock:      5,
		Lat:        f(30.5),
		Lng:        f(120.5),
	}
}

func TestValidateItemValid(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*ItemValidateRequest)
	}{
		{"baseline", func(r *ItemValidateRequest) {}},
		{"title_128_boundary", func(r *ItemValidateRequest) { r.Title = strings.Repeat("长", 128) }},
		{"deposit_zero", func(r *ItemValidateRequest) { r.Deposit = 0 }},
		{"daily_price_min", func(r *ItemValidateRequest) { r.DailyPrice = 0.01 }},
		{"stock_min", func(r *ItemValidateRequest) { r.Stock = 1 }},
		{"stock_max", func(r *ItemValidateRequest) { r.Stock = 999 }},
		{"lat_min", func(r *ItemValidateRequest) { r.Lat = f(-90) }},
		{"lat_max", func(r *ItemValidateRequest) { r.Lat = f(90) }},
		{"lng_min", func(r *ItemValidateRequest) { r.Lng = f(-180) }},
		{"lng_max", func(r *ItemValidateRequest) { r.Lng = f(180) }},
		{"lat_lng_not_passed", func(r *ItemValidateRequest) { r.Lat, r.Lng = nil, nil }},
		{"images_empty", func(r *ItemValidateRequest) { r.Images = "" }},
	}
	for _, tc := range cases {
		r := validItemReq()
		tc.mut(&r)
		if err := ValidateItem(r); err != nil {
			t.Errorf("%s: 应通过校验, got %v", tc.name, err)
		}
	}
}

func TestValidateItemReject(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*ItemValidateRequest)
	}{
		{"title_empty", func(r *ItemValidateRequest) { r.Title = "" }},
		{"title_blank", func(r *ItemValidateRequest) { r.Title = "   " }},
		{"title_too_long", func(r *ItemValidateRequest) { r.Title = strings.Repeat("长", 129) }},
		{"daily_price_zero", func(r *ItemValidateRequest) { r.DailyPrice = 0 }},
		{"daily_price_negative", func(r *ItemValidateRequest) { r.DailyPrice = -1 }},
		{"deposit_negative", func(r *ItemValidateRequest) { r.Deposit = -0.01 }},
		{"stock_zero", func(r *ItemValidateRequest) { r.Stock = 0 }},
		{"stock_negative", func(r *ItemValidateRequest) { r.Stock = -3 }},
		{"stock_1000", func(r *ItemValidateRequest) { r.Stock = 1000 }},
		{"lat_91", func(r *ItemValidateRequest) { r.Lat = f(91) }},
		{"lat_minus_91", func(r *ItemValidateRequest) { r.Lat = f(-91) }},
		{"lng_181", func(r *ItemValidateRequest) { r.Lng = f(181) }},
		{"lng_minus_181", func(r *ItemValidateRequest) { r.Lng = f(-181) }},
		{"images_not_array", func(r *ItemValidateRequest) { r.Images = "not-json" }},
		{"images_too_many", func(r *ItemValidateRequest) { r.Images = `["a","b","c","d","e","f","g","h","i","j"]` }},
		{"images_empty_url", func(r *ItemValidateRequest) { r.Images = `["https://a.com/x.jpg", ""]` }},
	}
	for _, tc := range cases {
		r := validItemReq()
		tc.mut(&r)
		if err := ValidateItem(r); err == nil {
			t.Errorf("%s: 应校验失败", tc.name)
		}
	}
}