// Package errs 存放跨领域的通用领域错误标记。
// 各具体领域（admin/order/...）的哨兵错误定义在各自的 domain/<领域>/ 包内，
// 这里只承载「无论哪个领域都会发生」的基础设施级错误。
//
// 用途：infrastructure 层把底层错误（gorm/mysql/redis 等）包装为
// StorageError 返回，application.CodeOf 便能将一切存储故障统一翻译为
// code.ErrorDatabase 的多语言文案，避免把 SQL 串等内幕直接漏给用户。
package errs

import "errors"

// ErrStorage 存储层发生错误。errors.Is(err, ErrStorage) 对所有经
// WrapStorage 包装的错误成立（含链上原错误，例如 gorm.ErrDuplicate 等）。
var ErrStorage = errors.New("存储层错误")

// StorageError 携带操作名与底层错误的存储层错误。
type StorageError struct {
	Op  string // 语义操作名，如 "admin.create"、"admin.list"
	Err error  // 底层错误（gorm / driver 等）
}

// Error 组装开发排查用文案（面向日志，不作为用户提示；用户文案走 i18n）。
func (e *StorageError) Error() string { return e.Op + ": " + e.Err.Error() }

// Unwrap 暴露底层错误，errors.Is / errors.As 逐层可用。
func (e *StorageError) Unwrap() error { return e.Err }

// Is 使 errors.Is(err, errs.ErrStorage) 对任意 StorageError 成立，
// 底层错误仍可继续通过 Unwrap 判别。
func (e *StorageError) Is(target error) bool { return target == ErrStorage }

// WrapStorage 给底层错误打上存储层标记；op 建议用 "领域.动作" 命名便于日志定位。
func WrapStorage(op string, err error) error {
	if err == nil {
		return nil
	}
	var se *StorageError
	if errors.As(err, &se) { // 已打标的不重复嵌套
		return &StorageError{Op: op + "." + se.Op, Err: se.Err}
	}
	return &StorageError{Op: op, Err: err}
}
