// Package application —— 基础设施层错误的码映射。
// 任意领域经 errs.WrapStorage 打标后统一落到 code.ErrorDatabase（40001），
// i18n 文案：zh "数据库操作出错,请重试" / en "database error, please retry"。
package application

import (
	"errors"

	"demo/app/biz/domain/errs"
	"demo/pkg/code"
)

func init() {
	RegisterErrMapper(func(err error) (int, bool) {
		// 存储层故障：无论源自哪个领域（admin/order/...），一律 40001；
		// 链上的领域业务错误（如 ErrUserNameTaken）会先被各自领域 mapper 命中，
		// 注册顺序由文件名（errors_admin.go < errors_infra.go）保证。
		if errors.Is(err, errs.ErrStorage) {
			return code.ErrorDatabase, true
		}
		return 0, false
	})
}
