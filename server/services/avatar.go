// 头像服务：上传校验（纯函数，无 DB 依赖，可离线单测）
package services

import (
	"errors"
	"path/filepath"
	"strings"
)

// AvatarMaxBytes 头像大小上限（4MB，低于 WAF body 上限 10MB）
const AvatarMaxBytes = 4 << 20

// ValidateAvatarName 校验头像文件名扩展名，返回规范化扩展名（小写、无点）。
func ValidateAvatarName(filename string) (string, error) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	switch ext {
	case "jpg", "jpeg", "png", "webp":
		return ext, nil
	}
	return "", errors.New("仅支持 jpg/jpeg/png/webp 图片")
}
