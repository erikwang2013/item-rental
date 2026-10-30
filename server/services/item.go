// 物品服务：输入校验（纯函数，无 DB 依赖，可离线单测）
package services

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"
)

// ItemValidateRequest 物品创建/更新的可校验字段。
type ItemValidateRequest struct {
	Title      string
	Desc       string
	Images     string
	DailyPrice float64
	Stock      int
	City       string
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
	// city 列宽 varchar(64)，超长直接 500，故在此拦截
	if utf8.RuneCountInString(req.City) > 64 {
		return errors.New("城市过长（最多 64 字）")
	}
	// desc/images 为 TEXT(65535 字节)，留余量后在此拦截，避免 DB 1406 → 500
	if len(req.Desc) > 60000 {
		return errors.New("描述过长（最多 60000 字节）")
	}
	if req.DailyPrice <= 0 {
		return errors.New("租金必须大于 0")
	}
	if req.Stock < 1 || req.Stock > 999 {
		return errors.New("库存必须在 1~999 之间")
	}
	if err := validateImages(req.Images); err != nil {
		return err
	}
	return nil
}

// validateImages 校验 Images 为 JSON 字符串数组（元素为 URL），长度 ≤9。
func validateImages(images string) error {
	if strings.TrimSpace(images) == "" {
		return nil
	}
	if len(images) > 60000 {
		return errors.New("图片列表过长（最多 60000 字节）")
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
		if len(u) > 2048 {
			return errors.New("图片 URL 过长（最多 2048 字节）")
		}
	}
	return nil
}
