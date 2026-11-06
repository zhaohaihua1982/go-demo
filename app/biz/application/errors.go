// Package application 应用层：编排领域逻辑，负责把领域错误转成业务错误码与文案。
//
// 错误码映射不集中在一个 switch：每个领域拥有自己的映射文件
// （errors_admin.go / errors_order.go / ...），通过 init 向本包注册，
// 新增业务领域 = 新增一个文件，不需要改本文件。
//
// 并发语义：Go 规范保证所有包的 init 串行执行且先于任何用户代码，
// 因此"init 写、运行读"的映射表无需加锁；运行期注册属于误用，
// 在首次查询后封闭并 panic（fail-fast），而不是静默丢映射。
package application

import (
	"sync/atomic"

	"demo/pkg/code"
	"demo/pkg/i18n"
)

var (
	// errMappers 领域错误映射表。只在各包 init 阶段追加、运行期只读，
	// 满足 Go 内置的 happens-before 语义（init 先于 main），无需锁。
	errMappers []func(err error) (int, bool)
	// sealed 首次查询后置真；此后再注册视为误用并 panic。
	sealed atomic.Bool
)

// RegisterErrMapper 供各领域的 errors_*.go 在 init 中注册领域映射。
// 顺序按文件名注册序，先注册者先匹配；领域内哨兵错误互不相识，通常无歧义。
func RegisterErrMapper(mapper func(err error) (int, bool)) {
	if sealed.Load() {
		panic("application.RegisterErrMapper: 必须在各包 init 阶段注册，首次查询之后视为误用")
	}
	errMappers = append(errMappers, mapper)
}

// CodeOf 把领域错误映射为 pkg/code 业务错误码，adapter 层组装响应时使用。
// err 为 nil 视为成功；所有领域映射都不命中时返回 code.ERROR，永不返回 nil 码。
func CodeOf(err error) int {
	if err == nil {
		return code.SUCCESS
	}
	sealed.Store(true) // 幂等：首查封装注册期，Store 并发安全且无副作用
	for _, mapper := range errMappers {
		if codeNum, ok := mapper(err); ok {
			return codeNum
		}
	}
	return code.ERROR
}

// Message 给出面向用户的业务提示文案（内部已含错误码映射），
// lang 典型取值：i18n.LangZH / i18n.LangEN / i18n.Default()。
func Message(err error, lang i18n.Lang) string {
	return i18n.Msg(lang, CodeOf(err))
}
