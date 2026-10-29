//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package win32

import (
	"encoding/binary"
	"testing"
)

// TestParseICO 解析 ICO 目录：条目偏移/长度正确，0 宽高按 256 处理，非法输入返回 nil。
func TestParseICO(t *testing.T) {
	payload := []byte{0x89, 'P', 'N', 'G', 1, 2, 3, 4}
	ico := make([]byte, 6+16+len(payload))
	binary.LittleEndian.PutUint16(ico[2:], 1) // type = icon
	binary.LittleEndian.PutUint16(ico[4:], 1) // count = 1
	e := ico[6:]
	e[0], e[1] = 0, 0 // 0 = 256px
	binary.LittleEndian.PutUint16(e[4:], 1)
	binary.LittleEndian.PutUint16(e[6:], 32)
	binary.LittleEndian.PutUint32(e[8:], uint32(len(payload)))
	binary.LittleEndian.PutUint32(e[12:], 22) // 6 + 16
	copy(ico[22:], payload)

	imgs := parseICO(ico)
	if len(imgs) != 1 {
		t.Fatalf("应解析出 1 个条目，实际 %d", len(imgs))
	}
	if imgs[0].size != 256 {
		t.Fatalf("0 宽高应视为 256，实际 %d", imgs[0].size)
	}
	if string(imgs[0].data) != string(payload) {
		t.Fatalf("图标数据不一致: %v", imgs[0].data)
	}

	if got := parseICO([]byte{1, 2, 3}); got != nil {
		t.Fatal("非法数据应返回 nil")
	}
	if got := parseICO(nil); got != nil {
		t.Fatal("空数据应返回 nil")
	}

	// 偏移越界的条目应被跳过
	bad := make([]byte, len(ico))
	copy(bad, ico)
	binary.LittleEndian.PutUint32(bad[6+12:], 9999)
	if got := parseICO(bad); len(got) != 0 {
		t.Fatalf("越界条目应被跳过，实际 %d", len(got))
	}
}
