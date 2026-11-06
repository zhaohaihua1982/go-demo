// Package application —— admin 领域的错误码映射。
// 新领域参照本文件：errors_<domain>.go + init 注册，不改 errors.go。
package application

import (
	"errors"

	admin "demo/app/biz/domain/admin"
	"demo/pkg/code"
)

func init() {
	RegisterErrMapper(func(err error) (int, bool) {
		switch {
		case errors.Is(err, admin.ErrUserNameTaken):
			return code.ErrorAdminNameTaken, true
		case errors.Is(err, admin.ErrNotFound):
			return code.ErrorAdminNotFound, true
		default:
			return 0, false // 不属于 admin 领域
		}
	})
}
