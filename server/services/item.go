// 物品服务：输入校验（纯函数，无 DB 依赖，可离线单测）
package services

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"
)

// ItemValidateRequest 物品创建/更新的可校验字段。
// Lat/Lng 用指针：nil 表示请求未传该字段，仅传了才校验并允许覆写。
type ItemValidateRequest struct {
	Title      string
	Images     string
	DailyPrice float64
	Deposit    float64
	Stock      int
	Lat        *float64
	Lng        *float64
}

// ValidateItem 校验物品输入，返回首个不合规字段的错误信息。
// 注：CategoryId 存在性校验需查 DB，由 controller 层做，不在此纯函数内。
func ValidateItem(req ItemValidateRequest) error {
	if strings.TrimSpace(req.Title) == "" {
		return errors.New("标题不能为空")
	}
	if utf8.RuneCountInString(req.Title) > 128 {
		return errors.New("标题过长（最多 128 字）")
	}
	if req.DailyPrice <= 0 {
		return errors.New("租金必须大于 0")
	}
	if req.Deposit < 0 {
		return errors.New("押金不能为负")
	}
	if req.Stock < 1 || req.Stock > 999 {
		return errors.New("库存必须在 1~999 之间")
	}
	if err := validateImages(req.Images); err != nil {
		return err
	}
	if req.Lat != nil && (*req.Lat < -90 || *req.Lat > 90) {
		return errors.New("纬度必须在 -90~90 之间")
	}
	if req.Lng != nil && (*req.Lng < -180 || *req.Lng > 180) {
		return errors.New("经度必须在 -180~180 之间")
	}
	return nil
}

// validateImages 校验 Images 为 JSON 字符串数组（元素为 URL），长度 ≤9。
func validateImages(images string) error {
	if strings.TrimSpace(images) == "" {
		return nil
	}
	var urls []string
	if err := json.Unmarshal([]byte(images), &urls); err != nil {
		return errors.New("图片格式错误：应为 JSON 字符串数组")
	}
	if len(urls) > 9 {
		return errors.New("图片最多 9 张")
	}
	for _, u := range urls {
		if strings.TrimSpace(u) == "" {
			return errors.New("图片 URL 不能为空")
		}
	}
	return nil
}