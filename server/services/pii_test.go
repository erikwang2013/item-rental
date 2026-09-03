// PII 保护单元测试：哈希稳定性、加解密往返、篡改检测、密钥 fail-fast
package services

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// 测试用 64 位 hex 密钥（AES-256）
const piiTestKey = "5f2c9a1e8b3d4f6071a2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718"

func TestPhoneHashStable(t *testing.T) {
	h1 := PhoneHash("13800138000")
	h2 := PhoneHash("13800138000")
	if h1 != h2 {
		t.Fatalf("同号哈希应稳定: %s != %s", h1, h2)
	}
	if len(h1) != 64 {
		t.Fatalf("sha256 hex 应为 64 字符, got %d", len(h1))
	}
	if PhoneHash("13800138000") == PhoneHash("13800138001") {
		t.Fatalf("不同手机号哈希不应相同")
	}
}

func TestRealNameRoundtrip(t *testing.T) {
	t.Setenv("ITEM_RENTAL_PII_KEY", piiTestKey)
	cases := []string{"张三", "A Very Long English Name Over Thirty Chars!!", ""}
	for _, name := range cases {
		enc, err := EncryptRealName(name)
		if err != nil {
			t.Fatalf("EncryptRealName(%q): %v", name, err)
		}
		if name == "" {
			if enc != "" {
				t.Fatalf("空串应直通, got %q", enc)
			}
			continue
		}
		// 密文必须非明文且长度足够容纳 base64(nonce+ct+tag)
		if enc == name || strings.Contains(enc, name) {
			t.Fatalf("密文泄露明文: %q", enc)
		}
		dec, err := DecryptRealName(enc)
		if err != nil {
			t.Fatalf("DecryptRealName: %v", err)
		}
		if dec != name {
			t.Fatalf("往返不一致: %q != %q", dec, name)
		}
	}
}

func TestRealNameTamperFails(t *testing.T) {
	t.Setenv("ITEM_RENTAL_PII_KEY", piiTestKey)
	enc, err := EncryptRealName("张三")
	if err != nil {
		t.Fatal(err)
	}
	// 篡改密文任意字节 → 解密必须报错（GCM 认证失败）
	tampered := "A" + enc[1:]
	if _, err := DecryptRealName(tampered); err == nil {
		t.Fatalf("篡改密文应解密失败")
	}
	if _, err := DecryptRealName("not-base64!!!"); err == nil {
		t.Fatalf("非法 base64 应报错")
	}
}

func TestPiiKeyFailFast(t *testing.T) {
	// 无密钥/非法密钥 → panic（fail-fast，密钥配错必须响亮）
	// 注意：PiiKey 有包级缓存，本用例须先清缓存再逐场景验证
	defer func() {
		piiKey = nil // 清理缓存，避免影响后续用例
	}()
	piiKey = nil
	t.Setenv("ITEM_RENTAL_PII_KEY", "")
	if err := catchPanic(func() { PiiKey() }); err == nil {
		t.Fatalf("缺失密钥应 panic")
	}
	piiKey = nil
	t.Setenv("ITEM_RENTAL_PII_KEY", "not-hex-key")
	if err := catchPanic(func() { PiiKey() }); err == nil {
		t.Fatalf("非法密钥应 panic")
	}
}

func catchPanic(f func()) (err error) {
	defer func() {
		if v := recover(); v != nil {
			if e, ok := v.(error); ok {
				err = e
			} else {
				err = errors.New(fmt.Sprint(v))
			}
		}
	}()
	f()
	return nil
}
