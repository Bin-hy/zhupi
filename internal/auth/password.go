package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	// PasswordMinRunes 是密码最小长度（按字符计）。
	PasswordMinRunes = 8
	// PasswordMaxBytes 是 bcrypt 允许的最大输入长度。
	PasswordMaxBytes = 72

	bcryptCost = 12
)

// dummyHash 用于账号不存在时执行一次等价开销的比对，降低时序差异。
var dummyHash = sync.OnceValue(func() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("zhupi-dummy-password"), bcryptCost)
	if err != nil {
		return nil
	}
	return hash
})

// HashPassword 使用 bcrypt 生成密码哈希。
// 只强制 bcrypt 的 72 字节上限；业务密码策略由调用方用 ValidatePassword 校验。
func HashPassword(plain string) (string, error) {
	if len(plain) > PasswordMaxBytes {
		return "", errors.New("密码不能超过 72 字节")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword 校验明文密码是否匹配哈希。
func VerifyPassword(hash, plain string) bool {
	if hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// VerifyDummy 在账号不存在时消耗一次与真实校验相近的时间。
func VerifyDummy(plain string) {
	_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(plain))
}

// ValidatePassword 校验密码长度：至少 8 个字符，最多 72 字节。
func ValidatePassword(plain string) error {
	if utf8.RuneCountInString(plain) < PasswordMinRunes {
		return errors.New("密码至少需要 8 个字符")
	}
	if len(plain) > PasswordMaxBytes {
		return errors.New("密码不能超过 72 字节")
	}
	return nil
}

// GeneratePassword 生成 16 位 URL-safe 随机密码。
func GeneratePassword() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return strings.TrimRight(base64.RawURLEncoding.EncodeToString(buf), "="), nil
}
