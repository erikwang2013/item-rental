// 头像服务：上传校验与落盘（纯函数 + 本地文件，无 DB 依赖，可离线单测）
package services

import (
	"bufio"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// AvatarMaxBytes 头像大小上限（4MB）
const AvatarMaxBytes = 4 << 20

var (
	// ErrUploadTooLarge 上传文件超过大小上限
	ErrUploadTooLarge = errors.New("upload too large")
	// ErrUploadType 文件内容与扩展名不符（或扩展名不在白名单）
	ErrUploadType = errors.New("upload type mismatch")
)

// ValidateAvatarName 校验头像文件名扩展名，返回规范化扩展名（小写、无点）。
func ValidateAvatarName(filename string) (string, error) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	switch ext {
	case "jpg", "jpeg", "png", "webp":
		return ext, nil
	}
	return "", errors.New("仅支持 jpg/jpeg/png/webp 图片")
}

// SaveUpload 将上传文件写入 dir/name：限制 max 字节，并按 name 扩展名校验真实内容。
// 失败时不残留半成品文件；超限返回 ErrUploadTooLarge，内容不符返回 ErrUploadType。
func SaveUpload(fh *multipart.FileHeader, dir, name string, max int64) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// 先嗅探头部再复制：仅凭扩展名可被伪造图片内容绕过
	br := bufio.NewReader(src)
	head, err := br.Peek(512)
	if err != nil && err != io.EOF {
		return err
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
	if err := checkImageContent(ext, head); err != nil {
		return err
	}

	dst, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	n, err := io.Copy(dst, io.LimitReader(br, max+1))
	closeErr := dst.Close()
	if err != nil {
		_ = os.Remove(filepath.Join(dir, name))
		return err
	}
	if closeErr != nil {
		_ = os.Remove(filepath.Join(dir, name))
		return closeErr
	}
	if n > max {
		_ = os.Remove(filepath.Join(dir, name))
		return ErrUploadTooLarge
	}
	return nil
}

// checkImageContent 校验 http.DetectContentType 嗅探结果与扩展名一致。
// 兼容 jpg/jpeg 两种写法；gif、文本、空内容一律拒绝。
func checkImageContent(ext string, head []byte) error {
	detected := http.DetectContentType(head)
	switch ext {
	case "jpg", "jpeg":
		if detected == "image/jpeg" {
			return nil
		}
	case "png":
		if detected == "image/png" {
			return nil
		}
	case "webp":
		if detected == "image/webp" {
			return nil
		}
	}
	return ErrUploadType
}
