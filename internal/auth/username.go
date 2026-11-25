package auth

import "errors"

// ValidateUsername 校验本地管理员用户名：3–32 个字符，仅字母、数字、`.`、`_`、`-`。
func ValidateUsername(username string) error {
	if len(username) < 3 || len(username) > 32 {
		return errors.New("用户名需为 3–32 个字符")
	}
	for _, r := range username {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' ||
			r == '.' || r == '_' || r == '-' {
			continue
		}
		return errors.New("用户名只能包含字母、数字、.、_、-")
	}
	return nil
}
