// 用户头像上传：multipart 落盘 + 直写 avatar URL（需登录）
package controllers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/erikwang2013/item-rental/server/middleware"
	"github.com/erikwang2013/item-rental/server/models"
	"github.com/erikwang2013/item-rental/server/services"
)

// avatarDir 头像保存目录（相对进程 CWD=server/，与 main.go SetStaticPath("/static","static") 同基准）
const avatarDir = "static/uploads/avatars"

// UploadAvatar 上传头像（需登录，仅本人）
// POST /api/v1/user/avatar  multipart 字段 file（jpg/jpeg/png/webp，≤4MB）
// 成功后直接更新 user.avatar 并返回 {avatar: URL}，前端无需再走 PUT profile。
func (c *UserController) UploadAvatar() {
	uid, ok := middleware.GetUserID(c.Ctx)
	if !ok {
		c.Fail(401, "未登录")
		return
	}

	fh, hdr, err := c.GetFile("file")
	if err != nil {
		c.Fail(400, "缺少 file 字段")
		return
	}
	defer fh.Close()

	ext, err := services.ValidateAvatarName(hdr.Filename)
	if err != nil {
		c.Fail(400, err.Error())
		return
	}

	o := orm.NewOrm()
	user := models.User{Id: uid}
	if err := o.Read(&user); err != nil {
		c.Fail(404, "用户不存在")
		return
	}

	// 文件名自生成（不信任客户端文件名，杜绝路径穿越）
	name := fmt.Sprintf("%d_%d.%s", uid, time.Now().UnixMilli(), ext)
	if err := os.MkdirAll(avatarDir, 0o755); err != nil {
		c.Fail(500, "保存失败")
		return
	}
	dst, err := os.Create(filepath.Join(avatarDir, name))
	if err != nil {
		c.Fail(500, "保存失败")
		return
	}
	defer dst.Close()

	n, err := io.Copy(dst, io.LimitReader(fh, services.AvatarMaxBytes+1))
	if err != nil {
		_ = os.Remove(dst.Name())
		c.Fail(500, "保存失败")
		return
	}
	if n > services.AvatarMaxBytes {
		_ = os.Remove(dst.Name())
		c.Fail(400, "文件不能超过 4MB")
		return
	}

	url := avatarURL(c.Ctx.Request, "/static/uploads/avatars/"+name)
	user.Avatar = url
	if _, err := o.Update(&user, "avatar"); err != nil {
		_ = os.Remove(dst.Name())
		c.Fail(500, "更新失败")
		return
	}
	c.OK(map[string]any{"avatar": url})
}

// avatarURL 拼头像完整 URL（纯函数）。https 判定：直连 TLS 或反代 X-Forwarded-Proto:https；
// 其余默认 http。生产反代环境建议配 ITEM_RENTAL_TRUSTED_PROXY 由反代统一改写。
func avatarURL(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := r.Host
	if host == "" {
		host = "127.0.0.1:8080"
	}
	return scheme + "://" + host + path
}
