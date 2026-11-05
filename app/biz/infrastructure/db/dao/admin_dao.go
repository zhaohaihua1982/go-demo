// Package dao 提供基于 GORM 的仓储实现。
// PO 结构体带 gorm 标签，只在本包使用；对外一律转换为 domain 领域实体。
package dao

import (
	"context"
	"errors"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	admin "demo/app/biz/domain/admin"
)

// AdminPO admin 表的数据对象，字段与表结构一一对应。
type AdminPO struct {
	ID             int64          `gorm:"primaryKey;autoIncrement"`
	CreatedAt      time.Time      `gorm:"column:created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index"`
	UserName       string         `gorm:"column:user_name"`
	PasswordDigest string         `gorm:"column:password_digest"`
	Avatar         string         `gorm:"column:avatar"`
}

// TableName 指定表名。
func (AdminPO) TableName() string { return "admin" }

// toDomain PO → 领域实体。
func (p *AdminPO) toDomain() *admin.Admin {
	if p == nil {
		return nil
	}
	return &admin.Admin{
		ID:             p.ID,
		UserName:       p.UserName,
		PasswordDigest: p.PasswordDigest,
		Avatar:         p.Avatar,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

// toPO 领域实体 → PO。
func toPO(a *admin.Admin) *AdminPO {
	if a == nil {
		return nil
	}
	return &AdminPO{
		ID:             a.ID,
		UserName:       a.UserName,
		PasswordDigest: a.PasswordDigest,
		Avatar:         a.Avatar,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
}

// AdminDAO admin 表仓储实现。
type AdminDAO struct {
	db *gorm.DB
}

// NewAdminDAO 注入 MySQL 连接。
func NewAdminDAO(db *gorm.DB) *AdminDAO {
	return &AdminDAO{db: db}
}

// 编译期断言：AdminDAO 满足仓储接口。
var _ admin.Repository = (*AdminDAO)(nil)

// Create 新增管理员。
// user_name 做唯一校验：先按登录名预查，再依赖库表唯一索引 uk_admin_user_name 做并发兜底，
// 两路冲突都统一返回 admin.ErrUserNameTaken。
func (d *AdminDAO) Create(ctx context.Context, a *admin.Admin) (int64, error) {
	existing, err := d.GetByUserName(ctx, a.UserName)
	if err != nil {
		return 0, err
	}
	if existing != nil {
		return 0, admin.ErrUserNameTaken
	}

	po := toPO(a)
	if err := d.db.WithContext(ctx).Create(po).Error; err != nil {
		// 预查后到插入之间可能有并发写入，插入撞唯一索引时再兜一次
		if isDuplicateKey(err) {
			return 0, admin.ErrUserNameTaken
		}
		return 0, err
	}
	a.ID = po.ID
	a.CreatedAt = po.CreatedAt
	a.UpdatedAt = po.UpdatedAt
	return po.ID, nil
}

// isDuplicateKey 判断错误是否为 MySQL 唯一键冲突（错误码 1062）。
func isDuplicateKey(err error) bool {
	var mysqlErr *mysqldriver.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

// GetByID 按主键查询（GORM 自动过滤 deleted_at），未命中返回 nil 和 nil 错误。
func (d *AdminDAO) GetByID(ctx context.Context, id int64) (*admin.Admin, error) {
	var po AdminPO
	err := d.db.WithContext(ctx).First(&po, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return po.toDomain(), nil
}

// GetByUserName 按登录名查询，用于登录校验，未命中返回 nil 和 nil 错误。
func (d *AdminDAO) GetByUserName(ctx context.Context, userName string) (*admin.Admin, error) {
	var po AdminPO
	err := d.db.WithContext(ctx).First(&po, "user_name = ?", userName).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return po.toDomain(), nil
}

// Update 按 id 更新非零字段。真正在表上执行的是：
//
//	UPDATE admin SET user_name = ?, avatar = ? WHERE id = ? AND deleted_at IS NULL
//
// PasswordDigest 传零值时不会被更新；未命中返回 ErrNotFound，
// 改名撞唯一索引 uk_admin_user_name 返回 ErrUserNameTaken。
func (d *AdminDAO) Update(ctx context.Context, a *admin.Admin) error {
	res := d.db.WithContext(ctx).
		Model(&AdminPO{}).
		Where("id = ?", a.ID).
		Updates(AdminPO{
			UserName: a.UserName,
			Avatar:   a.Avatar,
		})
	if res.Error != nil {
		if isDuplicateKey(res.Error) {
			return admin.ErrUserNameTaken
		}
		return res.Error
	}
	if res.RowsAffected == 0 {
		return admin.ErrNotFound
	}
	return nil
}

// Delete 软删除（写 deleted_at），未命中返回 ErrNotFound。
func (d *AdminDAO) Delete(ctx context.Context, id int64) error {
	res := d.db.WithContext(ctx).Delete(&AdminPO{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return admin.ErrNotFound
	}
	return nil
}

// List 分页查询：Keyword 非空时按 user_name 模糊过滤，按 id 降序。
func (d *AdminDAO) List(ctx context.Context, q admin.ListQuery) (*admin.ListResult, error) {
	q.Normalize()

	cond := d.db.WithContext(ctx).Model(&AdminPO{})
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		cond = cond.Where("user_name LIKE ?", "%"+kw+"%")
	}

	var total int64
	if err := cond.Count(&total).Error; err != nil {
		return nil, err
	}

	var pos []AdminPO
	if err := cond.
		Order("id DESC").
		Limit(q.PageSize).
		Offset(q.Offset()).
		Find(&pos).Error; err != nil {
		return nil, err
	}

	items := make([]admin.Admin, 0, len(pos))
	for i := range pos {
		items = append(items, *pos[i].toDomain())
	}

	return &admin.ListResult{
		Items:    items,
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}
