// 物品图片上传：multipart 多文件落盘，返回 URL 数组（需登录）
package controllers

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/erikwang2013/item-rental/server/middleware"
	"github.com/erikwang2013/item-rental/server/services"
)

// itemImageDir 物品图片保存目录（相对进程 CWD=server/，与 avatarDir 同基准）
const itemImageDir = "static/uploads/items"

// maxItemImages 一次上传最多张数（对齐服务端 images JSON 数组 ≤9 校验）
const maxItemImages = 9

// Upload 上传物品图片（需登录）。POST /api/v1/items/upload
// multipart 字段 files（单/多 part 均可，前端逐张上传）；jpg/jpeg/png/webp，单张 ≤4MB。
// 全部成功返回 {urls:[...]}；任一张失败即清理已落盘文件后 400（不返回部分成功）。
func (c *ItemController) Upload() {
	uid, ok := middleware.GetUserID(c.Ctx)
	if !ok {
		c.Fail(401, "未登录")
		return
	}

	files, _ := c.GetFiles("files")
	if len(files) == 0 {
		c.Fail(400, "未选择图片")
		return
	}
	if len(files) > maxItemImages {
		c.Fail(400, "一次最多 9 张")
		return
	}

	if err := os.MkdirAll(itemImageDir, 0o755); err != nil {
		c.Fail(500, "保存失败")
		return
	}

	// 文件名自生成（不信任客户端文件名，杜绝路径穿越）；任一张失败整批清理
	ms := time.Now().UnixMilli()
	names := make([]string, 0, len(files))
	for i, fh := range files {
		ext, err := services.ValidateAvatarName(fh.Filename)
		if err != nil {
			c.cleanupItemImages(names)
			c.Fail(400, err.Error())
			return
		}
		src, err := fh.Open()
		if err != nil {
			c.cleanupItemImages(names)
			c.Fail(400, "读取上传文件失败")
			return
		}
		name := fmt.Sprintf("%d_%d_%d.%s", uid, ms, i, ext)
		dst, err := os.Create(filepath.Join(itemImageDir, name))
		if err != nil {
			src.Close()
			c.cleanupItemImages(names)
			c.Fail(500, "保存失败")
			return
		}
		n, err := io.Copy(dst, io.LimitReader(src, services.AvatarMaxBytes+1))
		src.Close()
		dst.Close()
		if err != nil {
			_ = os.Remove(dst.Name())
			c.cleanupItemImages(names)
			c.Fail(500, "保存失败")
			return
		}
		if n > services.AvatarMaxBytes {
			_ = os.Remove(dst.Name())
			c.cleanupItemImages(names)
			c.Fail(400, "单张图片不能超过 4MB")
			return
		}
		names = append(names, name)
	}

	urls := make([]string, 0, len(names))
	for _, n := range names {
		urls = append(urls, avatarURL(c.Ctx.Request, "/static/uploads/items/"+n))
	}
	c.OK(map[string]any{"urls": urls})
}

// cleanupItemImages 删除已落盘文件（整批失败时调用，避免残留）
func (c *ItemController) cleanupItemImages(names []string) {
	for _, n := range names {
		_ = os.Remove(filepath.Join(itemImageDir, n))
	}
}
