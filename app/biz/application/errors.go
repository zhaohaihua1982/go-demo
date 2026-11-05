// Package application 应用层：编排领域逻辑，负责把领域错误转成业务错误码与文案。
package application

import (
	"errors"

	admin "demo/app/biz/domain/admin"
	"demo/pkg/code"
	"demo/pkg/i18n"
)

// CodeOf 把领域错误映射为 pkg/code 业务错误码，adapter 层组装响应时使用。
// err 为 nil 视为成功；未登记的错误返回 code.ERROR。
func CodeOf(err error) int {
	switch {
	case err == nil:
		return code.SUCCESS
	case errors.Is(err, admin.ErrUserNameTaken):
		return code.ErrorAdminNameTaken
	case errors.Is(err, admin.ErrNotFound):
		return code.ErrorAdminNotFound
	default:
		return code.ERROR
	}
}

// Message 给出面向用户的业务提示文案（内部已含错误码映射），
// lang 典型取值：i18n.LangZH / i18n.LangEN / i18n.Default()。
func Message(err error, lang i18n.Lang) string {
	return i18n.Msg(lang, CodeOf(err))
}
