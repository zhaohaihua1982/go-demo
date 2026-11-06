package i18n

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	// 取值原则：只用内置语言的写法变体和**永不真实存在**的标签（zz 前缀），
	// 一旦写死 fr-FR/ja-JP 这类真实语言标签，将来注册后断言就会失效。
	cases := []struct {
		env  string
		want Lang
	}{
		{"", LangZH},
		{"zh-CN", LangZH},
		{"zh_CN", LangZH},
		{"zh-cn", LangZH},
		{"en", LangEN},
		{"en-US", LangEN},
		{"en_US", LangEN},
		{"EN-US", LangEN},
		{"zz-fake", LangZH}, // 不可识别回退中文
		{"  ", LangZH},
	}
	for _, c := range cases {
		t.Setenv("DEMO_LANG", c.env)
		if got := Default(); got != c.want {
			t.Errorf("DEMO_LANG=%q: Default()=%q want %q", c.env, got, c.want)
		}
	}
}

func TestMsgBundles(t *testing.T) {
	requireBundles(t)
	t.Cleanup(Reset) // 兜底：避免外部污染显式默认语言
	Reset()

	// 两包键集合必须一致，避免某语言漏翻
	zh, en := bundles[LangZH], bundles[LangEN]
	if !reflect.DeepEqual(mapKeys(zh), mapKeys(en)) {
		t.Errorf("zh-CN 与 en-US 键集合不一致 zhOnly=%v enOnly=%v",
			diffKeys(zh, en), diffKeys(en, zh))
	}

	cases := []struct {
		lang Lang
		code int
		want string
	}{
		{LangZH, 200, "ok"},
		{LangEN, 201, "password changed successfully"},
		{LangZH, 30011, "管理员用户名已存在"},
		{LangEN, 30011, "admin username already exists"},
		{LangZH, 30012, "管理员不存在"},
	}
	for _, c := range cases {
		if got := Msg(c.lang, c.code); got != c.want {
			t.Errorf("Msg(%s, %d)=%q want %q", c.lang, c.code, got, c.want)
		}
	}

	// 未登记错误码回退 ERROR("500") 文案
	if got := Msg(LangZH, 99999); got != zh["500"] {
		t.Errorf("未登记错误码应回退 500 文案, got %q", got)
	}
	// 未知语言回退中文
	if got := Msg("zz-fake", 30011); got != "管理员用户名已存在" {
		t.Errorf("未知语言应回退中文, got %q", got)
	}
}

// diffKeys 返回 a 中存在而 b 中缺失的键。
func diffKeys(a, b map[string]string) []string {
	var d []string
	for k := range a {
		if _, ok := b[k]; !ok {
			d = append(d, k)
		}
	}
	return d
}

func TestPluggableRegistry(t *testing.T) {
	requireBundles(t)

	// 注册表应至少包含内置语言；未来加语言应自动出现，不做"全集相等"断言
	langs := Languages()
	byLang := Registered()
	if !sort.SliceIsSorted(langs, func(i, j int) bool { return langs[i] < langs[j] }) {
		t.Errorf("Languages() 应有序: %v", langs)
	}
	for _, want := range []Lang{LangZH, LangEN} {
		if _, ok := byLang[want]; !ok {
			t.Errorf("内置语言 %s 未注册", want)
		}
	}

	// 断言"未注册"用动态推导的标签（任何一门新语言加入都不至于改变测试语义）
	if Supported(unregisteredLang()) {
		t.Errorf("未注册标签 %q 不应视为已注册", unregisteredLang())
	}
	// ParseLang 严格语义：写法归一化可命中；裸子语言码不算注册
	if l, ok := ParseLang("EN_us"); !ok || l != LangEN {
		t.Errorf(`ParseLang("EN_us")=(%q,%v) want (en-US,true)`, l, ok)
	}
	if _, ok := ParseLang("en"); ok {
		t.Error(`裸 "en" 不应被视为已注册（明确文案应写 en-US）`)
	}
}

// unregisteredLang 推导一个当前必然未注册的语言标签。
// 测试不得点名具体未来语言（如 ja-JP），否则加入新语言时会误伤断言。
func unregisteredLang() Lang {
	reg := Registered()
	prim := make(map[string]bool, len(reg))
	for l := range reg {
		prim[strings.SplitN(normalize(string(l)), "-", 2)[0]] = true
	}
	for c := 'a'; c <= 'z'; c++ {
		p := string(c)
		if !prim[p] {
			return Lang(p + "-test")
		}
	}
	return "zzz-none-lang" // 26 个主子语言全被注册的极端情况
}

func TestSetDefaultLangObeysPriority(t *testing.T) {
	requireBundles(t)
	t.Cleanup(Reset)
	Reset()

	// 显式设置优先于环境变量
	t.Setenv("DEMO_LANG", "zh-CN")
	SetDefaultLang(LangEN)
	if got := Default(); got != LangEN {
		t.Errorf("SetDefaultLang 应优先于 DEMO_LANG, got %q", got)
	}

	// Msg 未指定已注册语言时按显式默认出文案
	if got := Msg(unregisteredLang(), 30011); got != "admin username already exists" {
		t.Errorf("未注册语言应回退显式默认语言, got %q", got)
	}

	// Reset 后回到环境变量
	Reset()
	if got := Default(); got != LangZH {
		t.Errorf("Reset 后应按 DEMO_LANG 解析, got %q", got)
	}
}

func TestSetDefaultLangRejectsUnregistered(t *testing.T) {
	requireBundles(t)
	t.Cleanup(Reset)
	Reset()
	t.Setenv("DEMO_LANG", "") // 密封：不受宿主环境影响

	SetDefaultLang(unregisteredLang()) // 未注册：应被拒绝，不改变默认语言
	if got := Default(); got != LangZH {
		t.Errorf("未注册语言设置后 Default 仍应回退链, got %q", got)
	}
}

// requireBundles 检查两个语言包都已成功解析。
func requireBundles(t *testing.T) {
	t.Helper()
	load()
	if len(bundles[LangZH]) == 0 || len(bundles[LangEN]) == 0 {
		t.Fatal("语言包未加载：locales 下应存在 zh-CN.json 与 en-US.json")
	}
}

func mapKeys(m map[string]string) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
