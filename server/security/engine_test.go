// 安全引擎纯函数测试：Content-Type 白名单 CSV 解析
package security

import (
	"reflect"
	"testing"
)

func TestSplitCSV(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"a, b ,, c", []string{"a", "b", "c"}}, // 去空白与空项
		{"", nil},
		{",,", nil},
	}
	for _, c := range cases {
		got := splitCSV(c.in)
		if c.want == nil {
			if len(got) != 0 {
				t.Errorf("splitCSV(%q) = %v, want empty", c.in, got)
			}
		} else if !reflect.DeepEqual(got, c.want) {
			t.Errorf("splitCSV(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}