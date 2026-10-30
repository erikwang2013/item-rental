// 头像校验单元测试：纯函数 + 本地临时目录，无 DB 依赖，可离线运行
package services

import (
	"bytes"
	"errors"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"
)

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

func TestCheckImageContent(t *testing.T) {
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}
	png := []byte("\x89PNG\r\n\x1a\nrest")
	webp := []byte("RIFF\x00\x00\x00\x00WEBPVP8 ")
	gif := []byte("GIF89a......")
	text := []byte("not an image at all")

	valid := []struct {
		ext  string
		head []byte
	}{
		{"jpg", jpeg}, {"jpeg", jpeg}, {"png", png}, {"webp", webp},
	}
	for _, tc := range valid {
		if err := checkImageContent(tc.ext, tc.head); err != nil {
			t.Errorf("checkImageContent(%q) 应通过, got %v", tc.ext, err)
		}
	}

	invalid := []struct {
		name string
		ext  string
		head []byte
	}{
		{"ext_mismatch", "png", jpeg},
		{"gif_ext", "gif", gif},
		{"gif_as_png", "png", gif},
		{"text_as_jpg", "jpg", text},
		{"empty", "png", nil},
	}
	for _, tc := range invalid {
		if err := checkImageContent(tc.ext, tc.head); !errors.Is(err, ErrUploadType) {
			t.Errorf("%s: 应返回 ErrUploadType, got %v", tc.name, err)
		}
	}
}

func TestSaveUpload(t *testing.T) {
	dir := t.TempDir()
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)

	if err := SaveUpload(fileHeader(t, "a.png", png), dir, "ok.png", int64(len(png))); err != nil {
		t.Fatalf("合法 png 应保存成功: %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "ok.png")); err != nil || !bytes.Equal(b, png) {
		t.Errorf("落盘内容不一致: %v", err)
	}

	if err := SaveUpload(fileHeader(t, "b.png", []byte("plain text")), dir, "spoof.png", 1024); !errors.Is(err, ErrUploadType) {
		t.Errorf("伪造扩展名应返回 ErrUploadType, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "spoof.png")); !os.IsNotExist(err) {
		t.Error("内容校验失败不应残留文件")
	}

	if err := SaveUpload(fileHeader(t, "c.png", png), dir, "big.png", 8); !errors.Is(err, ErrUploadTooLarge) {
		t.Errorf("超限应返回 ErrUploadTooLarge, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "big.png")); !os.IsNotExist(err) {
		t.Error("超限不应残留文件")
	}
}

// fileHeader 构造 multipart.FileHeader，供 SaveUpload 单测使用
func fileHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	form, err := multipart.NewReader(&buf, mw.Boundary()).ReadForm(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = form.RemoveAll() })
	return form.File["file"][0]
}
