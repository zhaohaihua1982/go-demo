package util

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestSumByString(t *testing.T) {
	// 定义一些常用的测试前置数据
	zero := decimal.Zero
	negativeOne := decimal.NewFromInt(-1)
	ten := decimal.NewFromInt(10)

	// 定义表驱动测试用例
	tests := []struct {
		name         string          // 测试用例名称
		a            string          // 参数 a
		b            string          // 参数 b
		defaultValue decimal.Decimal // 默认值
		want         string          // 期望的结果（用字符串表示，方便比对）
	}{
		{
			name:         "正常相加 (正数)",
			a:            "10.5",
			b:            "2.3",
			defaultValue: zero,
			want:         "12.8",
		},
		{
			name:         "正常相加 (包含负数)",
			a:            "-5.5",
			b:            "2.5",
			defaultValue: zero,
			want:         "-3",
		},
		{
			name:         "a 解析失败，使用默认值 0",
			a:            "abc",
			b:            "5",
			defaultValue: zero,
			want:         "5",
		},
		{
			name:         "b 解析失败，使用默认值 0",
			a:            "10",
			b:            "not-a-number",
			defaultValue: zero,
			want:         "10",
		},
		{
			name:         "a 和 b 都解析失败，使用默认值 0",
			a:            "error",
			b:            "error",
			defaultValue: zero,
			want:         "0",
		},
		{
			name:         "a 解析失败，使用自定义默认值 10",
			a:            "abc",
			b:            "5",
			defaultValue: ten,
			want:         "15",
		},
		{
			name:         "a 为空字符串（解析失败），使用负数默认值",
			a:            "",
			b:            "100",
			defaultValue: negativeOne,
			want:         "99",
		},
		{
			name:         "带空格和科学计数法的正常字符串",
			a:            " 1e3 ", // 虽然 decimal 会忽略前导/尾随空格，但以此测试其鲁棒性
			b:            "0.5",
			defaultValue: zero,
			want:         "1000.5",
		},
	}

	// 遍历执行测试
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 执行函数
			got := sumByString(tt.a, tt.b, tt.defaultValue)

			// 将结果转为 string 进行比对（因为 decimal 的 Equal 方法推荐这样用）
			// 注意：这里如果期望值带有小数点（如 "12.8"），got.String() 也会是 "12.8"
			if got.String() != tt.want {
				t.Errorf("sumByString(%q, %q, %s) = %s, want %s",
					tt.a, tt.b, tt.defaultValue.String(), got.String(), tt.want)
			}
		})
	}
}
