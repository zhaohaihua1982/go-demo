package code

import "demo/pkg/i18n"

// GetMsg 获取业务码对应的默认语言文案。
// 需要指定语言的场景（如按请求头 Accept-Language 应答）请直接使用 i18n.Msg。
func GetMsg(code int) string {
	return i18n.Msg(i18n.Default(), code)
}
