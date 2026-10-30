// 物品输入校验单元测试：纯函数，无 DB 依赖，可离线运行
package services

import (
	"encoding/json"
	"strings"
	"testing"
)

func validItemReq() ItemValidateRequest {
	return ItemValidateRequest{
		Title:      "相机",
		Desc:       "九成新",
		Images:     `["https://example.com/a.jpg"]`,
		DailyPrice: 50,
		Stock:      5,
		City:       "北京",
	}
}

// imagesWithURL 构造仅含一个 URL 的 images JSON 串
func imagesWithURL(url string) string {
	b, _ := json.Marshal([]string{url})
	return string(b)
}

// imagesOfTotalLen 构造总长度恰为 total 字节、含 n 个 URL 的 images JSON 串
func imagesOfTotalLen(t *testing.T, n, total int) string {
	t.Helper()
	urls := make([]string, n)
	for i := range urls {
		urls[i] = "https://example.com/" + strings.Repeat("a", 2048-len("https://example.com/"))
	}
	// json.Marshal 无空格：总长 = 各 URL 长度和 + 每个引号 2n + 逗号 (n-1) + 方括号 2
	extra := total - (n*2048 + 3*n + 1)
	last := len(urls[n-1]) + extra
	if last <= 0 || last > 2048 {
		t.Fatalf("total=%d 无法构造合法 URL 长度: %d", total, last)
	}
	urls[n-1] = urls[n-1][:last]
	b, err := json.Marshal(urls)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != total {
		t.Fatalf("构造长度 %d, 期望 %d", len(b), total)
	}
	return string(b)
}

func TestValidateItemValid(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*ItemValidateRequest)
	}{
		{"baseline", func(r *ItemValidateRequest) {}},
		{"title_128_boundary", func(r *ItemValidateRequest) { r.Title = strings.Repeat("长", 128) }},
		{"city_64_boundary", func(r *ItemValidateRequest) { r.City = strings.Repeat("城", 64) }},
		{"city_empty", func(r *ItemValidateRequest) { r.City = "" }},
		{"desc_60000_boundary", func(r *ItemValidateRequest) { r.Desc = strings.Repeat("a", 60000) }},
		{"images_url_2048_boundary", func(r *ItemValidateRequest) {
			r.Images = imagesWithURL("https://a.com/" + strings.Repeat("a", 2048-len("https://a.com/")))
		}},
		// 9 张 × 2048 是合法上限（总长 18460）；60000 总长在 ≤9 张约束下不可达
		{"images_max_valid", func(r *ItemValidateRequest) { r.Images = imagesOfTotalLen(t, 9, 18460) }},
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
		{"city_too_long", func(r *ItemValidateRequest) { r.City = strings.Repeat("城", 65) }},
		{"desc_too_long", func(r *ItemValidateRequest) { r.Desc = strings.Repeat("a", 60001) }},
		{"images_url_too_long", func(r *ItemValidateRequest) {
			r.Images = imagesWithURL("https://a.com/" + strings.Repeat("a", 2049-len("https://a.com/")))
		}},
		{"images_total_too_long", func(r *ItemValidateRequest) { r.Images = imagesOfTotalLen(t, 30, 60001) }},
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
