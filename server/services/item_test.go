// 物品输入校验单元测试：纯函数，无 DB 依赖，可离线运行
package services

import (
	"strings"
	"testing"
)

func validItemReq() ItemValidateRequest {
	return ItemValidateRequest{
		Title:      "相机",
		Images:     `["https://example.com/a.jpg"]`,
		DailyPrice: 50,
		Stock:      5,
	}
}

func TestValidateItemValid(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*ItemValidateRequest)
	}{
		{"baseline", func(r *ItemValidateRequest) {}},
		{"title_128_boundary", func(r *ItemValidateRequest) { r.Title = strings.Repeat("长", 128) }},
		{"daily_price_min", func(r *ItemValidateRequest) { r.DailyPrice = 0.01 }},
		{"stock_min", func(r *ItemValidateRequest) { r.Stock = 1 }},
		{"stock_max", func(r *ItemValidateRequest) { r.Stock = 999 }},
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
		{"stock_zero", func(r *ItemValidateRequest) { r.Stock = 0 }},
		{"stock_negative", func(r *ItemValidateRequest) { r.Stock = -3 }},
		{"stock_1000", func(r *ItemValidateRequest) { r.Stock = 1000 }},
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
