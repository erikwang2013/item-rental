// 头像校验单元测试：纯函数，无 DB 依赖，可离线运行
package services

import "testing"

func TestValidateAvatarName(t *testing.T) {
	valid := []string{"a.jpg", "a.JPG", "a.Jpeg", "a.png", "b.webp", "no_ext_dir/a.PnG"}
	for _, name := range valid {
		ext, err := ValidateAvatarName(name)
		if err != nil {
			t.Errorf("ValidateAvatarName(%q) 不应报错: %v", name, err)
			continue
		}
		if ext != "jpg" && ext != "png" && ext != "webp" && ext != "jpeg" {
			t.Errorf("ValidateAvatarName(%q) 扩展名 = %q, 应为小写白名单", name, ext)
		}
	}

	invalid := []string{"a.gif", "a.txt", "", "a", "a."}
	for _, name := range invalid {
		if _, err := ValidateAvatarName(name); err == nil {
			t.Errorf("ValidateAvatarName(%q) 应报错", name)
		}
	}
}
