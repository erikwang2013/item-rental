// PII 数据保护：手机号单向哈希（可等值查询）+ 姓名 AES-GCM 加密存储
package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"

	"github.com/beego/beego/v2/server/web"
)

// piiKey 密钥缓存（首次读取后复用；测试改 env 需在首个调用前 Setenv）
var piiKey []byte

// PiiKey 返回 PII 加密密钥：env ITEM_RENTAL_PII_KEY 优先，其次 app.conf pii_key。
// 必须为 64 位 hex（32 字节，AES-256）；缺失或非法直接 panic（fail-fast，密钥配错必须响亮，
// 镜像 middleware/jwt.go 的默认密钥防线）。
func PiiKey() []byte {
	if piiKey != nil {
		return piiKey
	}
	raw := os.Getenv("ITEM_RENTAL_PII_KEY")
	if raw == "" {
		raw = web.AppConfig.DefaultString("pii_key", "")
	}
	key, err := hex.DecodeString(raw)
	if err != nil || len(key) != 32 {
		panic("PII 密钥缺失或非法：请设置环境变量 ITEM_RENTAL_PII_KEY（64 位 hex，AES-256）")
	}
	piiKey = key
	return piiKey
}

// PhoneHash 手机号单向哈希（sha256 hex，64 字符）。users.phone 列存此值，
// 登录等值查询天然兼容；原始手机号仅存在于请求内存与短信通道，永不落库、永不下发。
func PhoneHash(phone string) string {
	sum := sha256.Sum256([]byte(phone))
	return hex.EncodeToString(sum[:])
}

// EncryptRealName 加密姓名：AES-256-GCM，输出 base64(nonce||ciphertext||tag)。空串直通。
func EncryptRealName(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	gcm, err := newGCM()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(s), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptRealName 解密姓名。空串直通（未实名用户）。
func DecryptRealName(b64 string) (string, error) {
	if b64 == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	gcm, err := newGCM()
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("密文长度非法")
	}
	nonce, ct := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func newGCM() (cipher.AEAD, error) {
	block, err := aes.NewCipher(PiiKey())
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
