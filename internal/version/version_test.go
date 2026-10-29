// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package version

import "testing"

type parseCase struct {
	in              string
	maj, min, patch int
	pre             string
	bad             bool
}

func TestParse(t *testing.T) {
	cases := []parseCase{
		{"1.2.3", 1, 2, 3, "", false},
		{"v1.2.3", 1, 2, 3, "", false},
		{" 0.1.0 ", 0, 1, 0, "", false},
		{"1.2", 1, 2, 0, "", false},
		{"1", 1, 0, 0, "", false},
		{"1.2.3-rc1", 1, 2, 3, "rc1", false},
		{"1.2.3-rc1+build.9", 1, 2, 3, "rc1", false},
		{"1.2.3+20261029", 1, 2, 3, "", false},
		{"", 0, 0, 0, "", true},
		{"1.2.3.4", 0, 0, 0, "", true},
		{"1..3", 0, 0, 0, "", true},
		{"a.b.c", 0, 0, 0, "", true},
	}
	for _, c := range cases {
		maj, min, patch, pre, err := Parse(c.in)
		if c.bad {
			if err == nil {
				t.Errorf("Parse(%q) 期望报错，实际返回 %d.%d.%d-%s", c.in, maj, min, patch, pre)
			}
			continue
		}
		if err != nil {
			t.Errorf("Parse(%q) 意外报错：%v", c.in, err)
			continue
		}
		if maj != c.maj || min != c.min || patch != c.patch || pre != c.pre {
			t.Errorf("Parse(%q) = %d.%d.%d-%q，期望 %d.%d.%d-%q",
				c.in, maj, min, patch, pre, c.maj, c.min, c.patch, c.pre)
		}
	}
}

type cmpCase struct {
	a, b string
	want int
}

func TestCompare(t *testing.T) {
	cases := []cmpCase{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.2.0", "1.10.0", -1},
		{"2.0.0", "1.99.99", 1},
		{"1.0.0-rc1", "1.0.0", -1},
		{"1.0.0", "1.0.0-rc1", 1},
		{"1.0.0-alpha", "1.0.0-beta", -1},
		{"v0.1.0", "0.1.0", 0},
	}
	for _, c := range cases {
		if got := Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%q, %q) = %d，期望 %d", c.a, c.b, got, c.want)
		}
	}
}

func TestCurrent(t *testing.T) {
	info := Current()
	if info.Product == "" || info.Number == "" {
		t.Fatalf("版本信息不完整：%+v", info)
	}
	if String() != "v"+Number {
		t.Errorf("String() = %q，期望 v%s", String(), Number)
	}
	if Title() != Product+" v"+Number {
		t.Errorf("Title() = %q", Title())
	}
	if _, _, _, _, err := Parse(Number); err != nil {
		t.Errorf("默认版本号 %q 不合法：%v", Number, err)
	}
}
