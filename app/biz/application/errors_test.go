package application_test

import (
	"fmt"
	"testing"

	"demo/app/biz/application"
	admin "demo/app/biz/domain/admin"
	"demo/pkg/i18n"
)

// TestMessageI18n 验证领域错误 → 业务码 → 双语文案的完整链路。
func TestMessageI18n(t *testing.T) {
	cases := []struct {
		name string
		err  error // 输入错误
		lang i18n.Lang
		code int
		msg  string
	}{
		{"用户名重复-中文", admin.ErrUserNameTaken, i18n.LangZH, 30011, "管理员用户名已存在"},
		{"用户名重复-英文", admin.ErrUserNameTaken, i18n.LangEN, 30011, "admin username already exists"},
		{"不存在-中文", admin.ErrNotFound, i18n.LangZH, 30012, "管理员不存在"},
		{"不存在-英文", admin.ErrNotFound, i18n.LangEN, 30012, "admin does not exist"},
		{"成功-中文", nil, i18n.LangZH, 200, "ok"},
		{"未登记错误回退ERROR", fmt.Errorf("任意错误"), i18n.LangEN, 500, "fail"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := application.CodeOf(c.err); got != c.code {
				t.Errorf("CodeOf=%d want %d", got, c.code)
			}
			if got := application.Message(c.err, c.lang); got != c.msg {
				t.Errorf("Message=%q want %q", got, c.msg)
			}
		})
	}
}

// TestMessageWithDefaultLang 固定 DEMO_LANG 的每种取值，验证默认语言解析语义。
// 内置语言（zh/en）断言确切文案；其余任意标签只断言**动态不变式**：
// 必须解析到某个已注册语言，且文案映射与该语言一致——
// 这样未来加入 ja-JP/fr-FR 等语言时测试无需改动即可继续成立。
func TestMessageWithDefaultLang(t *testing.T) {
	builtin := []struct {
		name     string
		env      string
		wantLang i18n.Lang
		wantMsg  string
	}{
		{"未设置默认中文", "", i18n.LangZH, "管理员用户名已存在"},
		{"zh-CN 显式设置", "zh-CN", i18n.LangZH, "管理员用户名已存在"},
		{"en-US 切英文", "en-US", i18n.LangEN, "admin username already exists"},
		{"裸 en 切英文", "en", i18n.LangEN, "admin username already exists"},
	}
	for _, c := range builtin {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("DEMO_LANG", c.env)

			gotLang := i18n.Default()
			if gotLang != c.wantLang {
				t.Errorf("DEMO_LANG=%q: Default()=%q want %q", c.env, gotLang, c.wantLang)
			}
			gotMsg := application.Message(admin.ErrUserNameTaken, gotLang)
			if gotMsg != c.wantMsg {
				t.Errorf("DEMO_LANG=%q: Message=%q want %q", c.env, gotMsg, c.wantMsg)
			}
		})
	}

	// 任意环境取值（含未注册/未来会注册的语言）的动态不变式
	invariants := []struct {
		name string
		env  string
	}{
		{"未注册标签回退解析", "zz-fake"},
		{"未来语言 ja-JP 无论注册与否都应正确解析", "ja-JP"},
		{"未来语言 fr-FR 无论注册与否都应正确解析", "fr-FR"},
	}
	for _, c := range invariants {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("DEMO_LANG", c.env)

			gotLang := i18n.Default()
			if !i18n.Supported(gotLang) {
				t.Fatalf("DEMO_LANG=%q: Default()=%q 必须解析到已注册语言或内置回退", c.env, gotLang)
			}
			gotMsg := application.Message(admin.ErrUserNameTaken, gotLang)
			if gotMsg == "" || gotMsg == "fail" {
				t.Errorf("DEMO_LANG=%q: 文案缺失或回退到 ERROR: %q", c.env, gotMsg)
			}
			// 映射一致性：Message 与 Msg 对同一解析语言结果相同
			if want := i18n.Msg(gotLang, 30011); gotMsg != want {
				t.Errorf("DEMO_LANG=%q: Message=%q 与 Msg(%s)=%q 不一致", c.env, gotMsg, gotLang, want)
			}
		})
	}
}

// TestMessageWithWrappedError 验证经 fmt.Errorf("%w") 包裹的领域错误仍能正确映射。
func TestMessageWithWrappedError(t *testing.T) {
	wrapped := fmt.Errorf("处理 admin 失败: %w", admin.ErrNotFound)
	if got := application.Message(wrapped, i18n.LangZH); got != "管理员不存在" {
		t.Errorf("包裹错误应映射到 30012 文案, got %q", got)
	}
}

// TestAdminErrMatrix 打印 admin 全部错误码的多语言对照（-v 可肉眼核对），
// 遍历**当前全部已注册语言**并断言文案不为空——未来加语言自动纳入覆盖。
func TestAdminErrMatrix(t *testing.T) {
	errs := []error{admin.ErrUserNameTaken, admin.ErrNotFound}
	for _, e := range errs {
		for _, lang := range i18n.Languages() {
			msg := application.Message(e, lang)
			if msg == "" || msg == "fail" {
				t.Errorf("错误 %v 在 %s 下文案缺失或回退到 ERROR: %q", e, lang, msg)
			}
			t.Logf("%v [%s] → code=%d msg=%q", e, lang, application.CodeOf(e), msg)
		}
	}
}
