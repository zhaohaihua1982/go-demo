// Package i18n 按业务错误码（pkg/code 中定义的数字码）提供多语言文案。
//
// 设计支持语言的"可插拔"：
//
//   - 新增一门语言：仅需在 locales/ 下放同名 json（如 ja-JP.json），
//     编译期内嵌后自动注册，无需改动任何 Go 代码；
//   - 切换默认语言：在 yaml 配置 i18n.defaultLang（由 bootstrap 在启动时应用），
//     或进程级环境变量 DEMO_LANG；显式配置优先于环境变量；
//   - 运行时切换：调用 SetDefaultLang（全局会话）；测试出口 Reset。
//
// 文案键为错误码的字符串形式（如 "30011"）。本包不 import pkg/code 以避免循环依赖。
package i18n

import (
	"embed"
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// Lang 语言标签，即 locales/ 下的文件名（不含扩展名）。
type Lang string

const (
	LangZH Lang = "zh-CN"
	LangEN Lang = "en-US"
)

// fallbackLang 任何一层配置都未命中时的最终回退语言。
const fallbackLang = LangZH

//go:embed locales
var localeFS embed.FS

var (
	once    sync.Once
	bundles map[Lang]map[string]string
	// defaultLang 显式设置的默认语言（来自 yaml 配置或 SetDefaultLang），
	// 指针为 nil 表示未显式设置。
	defaultLang atomic.Pointer[Lang]
)

// load 编译期内嵌数据：扫描 locales/ 目录注册全部语言包。
func load() {
	once.Do(func() {
		entries, err := localeFS.ReadDir("locales")
		if err != nil {
			return
		}
		bundles = make(map[Lang]map[string]string, len(entries))
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".json") {
				continue
			}
			data, err := localeFS.ReadFile("locales/" + name)
			if err != nil {
				continue
			}
			var m map[string]string
			if json.Unmarshal(data, &m) != nil {
				continue
			}
			bundles[Lang(strings.TrimSuffix(name, ".json"))] = m
		}
	})
}

// normalize 归一化语言写法：大小写不敏感、下划线视为连字符。
func normalize(s string) string {
	return strings.ToLower(
		strings.ReplaceAll(strings.TrimSpace(s), "_", "-"),
	)
}

// ParseLang 严格解析语言标签：必须命中已注册的语言包才算支持。
// "en-US" / "en_us" / "EN-US" 等写法均可；裸 "en" 属未注册语义，返回 false。
func ParseLang(s string) (Lang, bool) {
	norm := normalize(s)
	for lang := range Registered() {
		if norm == normalize(string(lang)) {
			return lang, true
		}
	}
	return "", false
}

// Registered 返回已注册的完整语言名集合（键为 Lang，值为包内条目数）。
func Registered() map[Lang]int {
	load()
	out := make(map[Lang]int, len(bundles))
	for l, b := range bundles {
		out[l] = len(b)
	}
	return out
}

// Languages 返回已注册语言的排序切片，供 adapter 列举可选值等场景使用。
func Languages() []Lang {
	load()
	langs := make([]Lang, 0, len(bundles))
	for l := range bundles {
		langs = append(langs, l)
	}
	sort.Slice(langs, func(i, j int) bool { return langs[i] < langs[j] })
	return langs
}

// Supported 判断语言包是否已注册。
func Supported(l Lang) bool { _, ok := ParseLang(string(l)); return ok }

// SetDefaultLang 显式设定默认语言（如启动时应用 yaml 的 i18n.defaultLang）。
// 设置后的优先级高于环境变量，直到 Reset。
func SetDefaultLang(l Lang) {
	load()
	if Supported(l) {
		defaultLang.Store(&l)
	}
}

// Reset 清除显式默认语言，退回 env 解析（主要用于测试隔离）。
func Reset() { defaultLang.Store(nil) }

// Default 解析默认语言，优先级：
//  1. SetDefaultLang 显式设置（yaml 配置在启动时应用到这里）
//  2. 环境变量 DEMO_LANG
//  3. 内置回退 fallbackLang
func Default() Lang {
	if p := defaultLang.Load(); p != nil {
		return *p
	}
	return fromEnv()
}

// fromEnv 按 DEMO_LANG 环境变量取语言，三层匹配：
//  1. 精确匹配已注册语言（en-us → en-US）
//  2. 变体前缀唯一命中（en-remoteIsland → en-US，仅 en 系已注册时）
//  3. 主子语言唯一命中（en → en-US）
//
// 均未命中（如 ja-JP）回退 fallbackLang。
func fromEnv() Lang {
	norm := normalize(os.Getenv("DEMO_LANG"))
	if norm == "" {
		return fallbackLang
	}
	langs := Languages()
	primaryOf := func(l Lang) string {
		return strings.SplitN(normalize(string(l)), "-", 2)[0]
	}

	var exact, variant, primary []Lang
	for _, l := range langs {
		nl := normalize(string(l))
		switch {
		case nl == norm:
			exact = append(exact, l)
		case strings.HasPrefix(nl, norm+"-"):
			variant = append(variant, l)
		case primaryOf(l) == strings.SplitN(norm, "-", 2)[0]:
			primary = append(primary, l)
		}
	}
	switch {
	case len(exact) == 1:
		return exact[0]
	case len(variant) == 1:
		return variant[0]
	case len(primary) == 1:
		return primary[0]
	}
	return fallbackLang
}

// Msg 按 lang 取错误码 bizCode 对应文案。
// lang 未注册时按顺序回退：显式默认语言 → 环境变量语言 → 任意已注册语言包；
// 文案缺失时回退该语言包内的 "500"（对应 code.ERROR 的文案）。
// 整个解析单遍完成，无递归，语言包缺失时返回空串。
func Msg(lang Lang, bizCode int) string {
	load()
	key := strconv.Itoa(bizCode)

	if b, ok := bundles[lang]; ok {
		return msgFrom(b, key)
	}

	if dl := defaultLang.Load(); dl != nil {
		if b, found := bundles[*dl]; found {
			return msgFrom(b, key)
		}
	}
	if b, found := bundles[fromEnv()]; found {
		return msgFrom(b, key)
	}
	for _, b := range bundles { // 最终兜底：任意一个已注册语言包
		return msgFrom(b, key)
	}
	return ""
}

// msgFrom 在单个语言包内取指定键文案，缺失时退 500。
func msgFrom(b map[string]string, key string) string {
	if msg, ok := b[key]; ok && msg != "" {
		return msg
	}
	return b["500"]
}
