// Package admin 定义管理员领域实体与仓储接口。
// 本包不依赖任何具体存储实现，接口由 infrastructure/db/dao 提供实现。
package admin

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound 仓储查不到目标记录时返回。
var ErrNotFound = errors.New("admin 不存在")

// ErrUserNameTaken 新增/改用户名时与已有记录冲突时返回。
var ErrUserNameTaken = errors.New("admin 用户名已存在")

// Admin 管理员领域实体。
type Admin struct {
	ID             int64
	UserName       string
	PasswordDigest string
	Avatar         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ListQuery 分页查询条件。Page 从 1 开始；Keyword 按 user_name 模糊匹配，可选。
type ListQuery struct {
	Page     int
	PageSize int
	Keyword  string
}

// Normalize 兜底修正非法分页参数。
func (q *ListQuery) Normalize() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 10
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
}

// Offset 计算查询偏移量。
func (q *ListQuery) Offset() int {
	q.Normalize()
	return (q.Page - 1) * q.PageSize
}

// ListResult 分页查询结果。
type ListResult struct {
	Items    []Admin
	Total    int64 // 满足条件的总记录数
	Page     int
	PageSize int
}

// Repository 管理员仓储接口，由 infrastructure/db/dao 实现。
type Repository interface {
	// Create 新增管理员。user_name 与已有记录冲突时返回 ErrUserNameTaken。
	Create(ctx context.Context, a *Admin) (int64, error)
	// GetByID 按主键查询，未命中返回 nil 和 nil 错误。
	GetByID(ctx context.Context, id int64) (*Admin, error)
	// GetByUserName 按登录名查询，未命中返回 nil 和 nil 错误。
	GetByUserName(ctx context.Context, userName string) (*Admin, error)
	// Update 按 id 更新实体的非零字段，未命中返回 ErrNotFound。
	Update(ctx context.Context, a *Admin) error
	// Delete 软删除，未命中返回 ErrNotFound。
	Delete(ctx context.Context, id int64) error
	// List 分页查询，Keyword 非空时按 user_name 模糊过滤。
	List(ctx context.Context, q ListQuery) (*ListResult, error)
}
