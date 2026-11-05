package dao_test

// 集成测试：依赖本地 MySQL（conf/locales/config.yaml 的 mysql.default 配置），
// 数据库不可连时自动 Skip，不影响其他环境跑测试。

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	admin "demo/app/biz/domain/admin"
	"demo/app/biz/infrastructure/db"
	"demo/app/biz/infrastructure/db/dao"
	"demo/conf"
)

func TestMain(m *testing.M) {
	if err := conf.InitConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "配置初始化失败:", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// setup 连库并返回 repository，连不上则跳过本测试。
func setup(t *testing.T) (admin.Repository, *gorm.DB) {
	t.Helper()
	gormDB, err := db.InitMySQL(conf.Config.MySql["default"], false)
	if err != nil {
		t.Skipf("本地 mysql 不可用，跳过集成测试: %v", err)
	}
	t.Cleanup(func() { _ = db.Close(gormDB) })
	return dao.NewAdminDAO(gormDB), gormDB
}

// uniqueName 生成测试专用用户名，纳秒级时间戳规避历史/并发数据冲突。
func uniqueName(prefix string) string {
	return fmt.Sprintf("it_%s_%d", prefix, time.Now().UnixNano())
}

// removeRow 测试结束无条件硬删（含软删除），保证幂等。
// 注意：必须先 WithContext 取会话再 Unscoped，根连接直接链式 Unscoped
// 会污染共享状态，导致后续删除退化为软删。
func removeRow(gormDB *gorm.DB, id int64) func() {
	return func() {
		gormDB.WithContext(context.Background()).Unscoped().Delete(&dao.AdminPO{}, id)
	}
}

func TestAdminCreateUniqueName(t *testing.T) {
	r, gormDB := setup(t)
	ctx := context.Background()
	name := uniqueName("dup")

	id, err := r.Create(ctx, &admin.Admin{UserName: name, PasswordDigest: "d1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(removeRow(gormDB, id))

	// 同名重复创建 → 预检路径命中 ErrUserNameTaken
	_, err = r.Create(ctx, &admin.Admin{UserName: name})
	if !errors.Is(err, admin.ErrUserNameTaken) {
		t.Fatalf("重复创建应返回 ErrUserNameTaken, got %v", err)
	}

	// 软删除后重建同名：预查不可见软删记录，应撞唯一索引并同样转成 ErrUserNameTaken
	if err := r.Delete(ctx, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = r.Create(ctx, &admin.Admin{UserName: name})
	if !errors.Is(err, admin.ErrUserNameTaken) {
		t.Fatalf("软删后重建应返回 ErrUserNameTaken, got %v", err)
	}
}

func TestAdminCreateUniqueNameConcurrent(t *testing.T) {
	// 并发同名创建只能成功一个，其余必须走唯一索引兜底返回 ErrUserNameTaken
	r, gormDB := setup(t)
	ctx := context.Background()
	name := uniqueName("race")

	const n = 8
	ids := make([]int64, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ids[i], errs[i] = r.Create(ctx, &admin.Admin{UserName: name})
		}(i)
	}
	wg.Wait()

	var okCount, taken int
	for i, err := range errs {
		switch {
		case err == nil:
			okCount++
			t.Cleanup(removeRow(gormDB, ids[i]))
		case errors.Is(err, admin.ErrUserNameTaken):
			taken++
		default:
			t.Fatalf("并发创建出现意外错误: %v", err)
		}
	}
	if okCount != 1 || taken != n-1 {
		t.Fatalf("并发结果不对：成功 %d 个、冲突 %d 个（期望 %d/%d）", okCount, taken, 1, n-1)
	}
}

func TestAdminUpdateRenameConflict(t *testing.T) {
	r, gormDB := setup(t)
	ctx := context.Background()
	nameA := uniqueName("ua")
	nameB := uniqueName("ub")

	idA, err := r.Create(ctx, &admin.Admin{UserName: nameA})
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	t.Cleanup(removeRow(gormDB, idA))
	idB, err := r.Create(ctx, &admin.Admin{UserName: nameB})
	if err != nil {
		t.Fatalf("create B: %v", err)
	}
	t.Cleanup(removeRow(gormDB, idB))

	// 把 B 改名成 A 的名字 → 撞唯一索引
	err = r.Update(ctx, &admin.Admin{ID: idB, UserName: nameA, Avatar: "http://x"})
	if !errors.Is(err, admin.ErrUserNameTaken) {
		t.Fatalf("改名撞库应返回 ErrUserNameTaken, got %v", err)
	}

	// 撞库后 B 的原数据不应被破坏
	gotB, err := r.GetByID(ctx, idB)
	if err != nil || gotB == nil {
		t.Fatalf("get B: err=%v got=%v", err, gotB)
	}
	if gotB.UserName != nameB {
		t.Fatalf("撞库写入失败后 user_name 应仍是 %q, got %q", nameB, gotB.UserName)
	}
}

func TestAdminUpdateNotFound(t *testing.T) {
	r, _ := setup(t)
	err := r.Update(context.Background(), &admin.Admin{ID: -1, UserName: uniqueName("nope")})
	if !errors.Is(err, admin.ErrNotFound) {
		t.Fatalf("更新不存在记录应返回 ErrNotFound, got %v", err)
	}
}

func TestAdminGetNotFound(t *testing.T) {
	r, _ := setup(t)
	if got, err := r.GetByID(context.Background(), -1); got != nil || err != nil {
		t.Fatalf("GetByID 不存在应返回 (nil, nil), got (%v, %v)", got, err)
	}
	if got, err := r.GetByUserName(context.Background(), uniqueName("nouser")); got != nil || err != nil {
		t.Fatalf("GetByUserName 不存在应返回 (nil, nil), got (%v, %v)", got, err)
	}
}

func TestAdminDelete(t *testing.T) {
	r, gormDB := setup(t)
	ctx := context.Background()

	id, err := r.Create(ctx, &admin.Admin{UserName: uniqueName("del")})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(removeRow(gormDB, id))
	if err := r.Delete(ctx, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	// 已删不可再删
	if err := r.Delete(ctx, id); !errors.Is(err, admin.ErrNotFound) {
		t.Fatalf("重复删除应返回 ErrNotFound, got %v", err)
	}
	// 软删除后查不到
	got, err := r.GetByID(ctx, id)
	if got != nil || err != nil {
		t.Fatalf("软删除后应查不到, got (%v, %v)", got, err)
	}
}

func TestAdminListPaginationAndKeyword(t *testing.T) {
	r, gormDB := setup(t)
	ctx := context.Background()
	batch := uniqueName("lst")

	// 造 3 条 Keyword 可命中的数据
	ids := make([]int64, 0, 3)
	for _, suffix := range []string{"a", "b", "c"} {
		id, err := r.Create(ctx, &admin.Admin{UserName: batch + suffix})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		ids = append(ids, id)
		t.Cleanup(removeRow(gormDB, id))
	}

	resp, err := r.List(ctx, admin.ListQuery{Page: 1, PageSize: 2, Keyword: batch})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if resp.Total != 3 || len(resp.Items) != 2 {
		t.Fatalf("第 1 页应为 2/3 条, got total=%d len=%d", resp.Total, len(resp.Items))
	}
	if resp.Items[0].ID != ids[2] || resp.Items[1].ID != ids[1] {
		t.Fatalf("应按 id 降序: got %d,%d want %d,%d", resp.Items[0].ID, resp.Items[1].ID, ids[2], ids[1])
	}

	resp2, err := r.List(ctx, admin.ListQuery{Page: 2, PageSize: 2, Keyword: batch})
	if err != nil {
		t.Fatalf("list page2: %v", err)
	}
	if len(resp2.Items) != 1 || resp2.Items[0].ID != ids[0] {
		t.Fatalf("第 2 页应剩 1 条 id=%d, got %+v", ids[0], resp2.Items)
	}

	// 无匹配 Keyword
	resp3, err := r.List(ctx, admin.ListQuery{Keyword: uniqueName("none")})
	if err != nil {
		t.Fatalf("list none: %v", err)
	}
	if resp3.Total != 0 || len(resp3.Items) != 0 {
		t.Fatalf("无匹配应为空, got total=%d items=%d", resp3.Total, len(resp3.Items))
	}

	// 非法分页参数被修正
	q := admin.ListQuery{Page: -1, PageSize: 9999}
	q.Normalize()
	if off := q.Offset(); off != 0 {
		t.Fatalf("非法分页参数应被 Normalize 修正, got offset=%d", off)
	}
}
