// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package x11

import "testing"

// XEvent 字段偏移是这套绑定最容易出错的地方（结构体对齐、int/long 混排），
// 这里直接构造字节流断言解码结果，保证偏移量正确、且不依赖 X 服务器。

func TestEventType(t *testing.T) {
	var ev Event
	ev.SetField(0, uint64(ButtonPress))
	if got := ev.Type(); got != ButtonPress {
		t.Fatalf("Type() = %d, want %d", got, ButtonPress)
	}
}

func TestEventPointerCoords(t *testing.T) {
	var ev Event
	ev.SetField(0, uint64(MotionNotify))
	ev.SetField(64, uint64(int64(int32(321)))) // x（int32 写回）
	ev.SetField(68, uint64(int64(int32(654)))) // y
	if ev.X() != 321 || ev.Y() != 654 {
		t.Fatalf("X/Y = %d/%d, want 321/654", ev.X(), ev.Y())
	}
}

func TestEventNegativeCoords(t *testing.T) {
	var ev Event
	// 指针移出窗口时 X 可能给出负坐标（int32 → uint64 符号扩展）
	nx, ny := int32(-12), int32(-7)
	ev.SetField(64, uint64(int64(nx)))
	ev.SetField(68, uint64(int64(ny)))
	if ev.X() != -12 || ev.Y() != -7 {
		t.Fatalf("X/Y = %d/%d, want -12/-7", ev.X(), ev.Y())
	}
}

func TestEventButtonAndKeycode(t *testing.T) {
	var ev Event
	ev.SetField(84, 5) // button / keycode 共用偏移 84
	if ev.Button() != 5 {
		t.Fatalf("Button() = %d, want 5", ev.Button())
	}
	if ev.Keycode() != 5 {
		t.Fatalf("Keycode() = %d, want 5", ev.Keycode())
	}
}

func TestEventConfigureSize(t *testing.T) {
	var ev Event
	ev.SetField(0, uint64(ConfigureNotify))
	ev.SetField(56, uint64(int64(int32(1280)))) // width
	ev.SetField(60, uint64(int64(int32(720))))  // height
	if ev.Width() != 1280 || ev.Height() != 720 {
		t.Fatalf("W/H = %d/%d, want 1280/720", ev.Width(), ev.Height())
	}
}

func TestEventClientMessage(t *testing.T) {
	var ev Event
	ev.SetField(0, uint64(ClientMessage))
	ev.SetField(40, 0x1234) // message_type
	ev.SetField(56, 0xABC)  // data[0]
	ev.SetField(64, 0xDEF)  // data[1]
	if got := ev.MessageType(); got != 0x1234 {
		t.Fatalf("MessageType() = %#x, want 0x1234", got)
	}
	if got := ev.Data(0); got != 0xABC {
		t.Fatalf("Data(0) = %#x, want 0xABC", got)
	}
	if got := ev.Data(1); got != 0xDEF {
		t.Fatalf("Data(1) = %#x, want 0xDEF", got)
	}
}

func TestEventOutOfRangeIsSafe(t *testing.T) {
	var ev Event
	// 越界读取必须返回 0 而不是 panic（防止事件类型未知时读飞）
	if got := at(ev.Raw[:], len(ev.Raw)-4); got != 0 {
		t.Fatalf("越界读取 = %d, want 0", got)
	}
	// 越界写入同样安全
	setLong(ev.Raw[:], len(ev.Raw)-1, 1)
	ev.SetField(-8, 1)
}
