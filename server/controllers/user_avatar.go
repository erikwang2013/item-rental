// 用户头像上传：multipart 落盘 + 直写 avatar URL（需登录）
package controllers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	if err := services.SaveUpload(hdr, avatarDir, name, services.AvatarMaxBytes); err != nil {
		code, msg := uploadErr(err)
		c.Fail(code, msg)
		return
	}

	url := avatarURL(c.Ctx.Request, "/static/uploads/avatars/"+name)
	oldAvatar := user.Avatar
	user.Avatar = url
	if _, err := o.Update(&user, "avatar"); err != nil {
		_ = os.Remove(filepath.Join(avatarDir, name))
		c.Fail(500, "更新失败")
		return
	}
	deleteOldAvatar(uid, oldAvatar, url)
	c.OK(map[string]any{"avatar": url})
}

// deleteOldAvatar 尽力删除本人旧头像文件，删除失败忽略（不影响响应）。
func deleteOldAvatar(uid int64, oldURL, newURL string) {
	if name := localAvatarFileName(uid, oldURL, newURL); name != "" {
		_ = os.Remove(filepath.Join(avatarDir, name))
	}
}

// localAvatarFileName 返回可安全删除的本人本地头像文件名，否则返回空串。
// 仅接受本服务头像目录下、文件名为 "<uid>_" 开头的 URL（上传名格式 %d_%d.ext）；
// 外部 URL、他人文件、新旧相同、含路径分隔符或 . / .. 一律跳过。
func localAvatarFileName(uid int64, oldURL, newURL string) string {
	const marker = "/static/uploads/avatars/"
	if oldURL == newURL {
		return ""
	}
	i := strings.LastIndex(oldURL, marker)
	if i < 0 {
		return ""
	}
	base := oldURL[i+len(marker):]
	if base == "" || base == "." || base == ".." || strings.ContainsAny(base, `/\`) {
		return ""
	}
	if !strings.HasPrefix(base, strconv.FormatInt(uid, 10)+"_") {
		return ""
	}
	return base
}

// avatarURL 拼头像完整 URL（纯函数）。https 判定：直连 TLS 或反代 X-Forwarded-Proto:https；
// 其余默认 http。
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
