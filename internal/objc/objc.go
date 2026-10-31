//go:build darwin

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

// Package objc 极简 Objective-C 运行时桥（purego 动态加载，零 cgo）。
//
// 只做框架需要的事：查类 / 查选择子 / 发消息 / 动态加方法 / NSString 互转。
// AppKit 与 Foundation 用 dlopen 加载，因此不需要 cgo、也不需要链接参数。
//
// 关于结构体：Cocoa 大量按值传递 NSRect / NSPoint（2~4 个 CGFloat）。
// purego 在 darwin/amd64 与 darwin/arm64 上支持「调用 C 函数时按值传/回结构体」，
// 因此这里为常见参数组合各绑一个类型化函数（SendRect / SendRect3 …），
// 而不是用无类型的 SyscallN 硬拼寄存器。
//
// 注意：purego 的 NewCallback **不支持**结构体参数，因此在 Cocoa 侧我们只注入
// 「参数全是对象指针」的委托方法（mouseDown: / keyDown: …），
// 需要矩形/坐标时在回调内部另行发消息获取（这是普通 C 调用，支持结构体）。
package objc

import (
	"errors"
	"unsafe"

	"github.com/ebitengine/purego"
)

// Point NSPoint / CGPoint。
type Point struct{ X, Y float64 }

// Size NSSize / CGSize。
type Size struct{ W, H float64 }

// Rect NSRect / CGRect。
type Rect struct {
	Origin Point
	Size   Size
}

// NewRect 构造矩形。
func NewRect(x, y, w, h float64) Rect {
	return Rect{Origin: Point{X: x, Y: y}, Size: Size{W: w, H: h}}
}

var (
	loadErr error
	loaded  bool

	objcGetCls  func(string) uintptr
	selRegName  func(string) uintptr
	msgSendAddr uintptr
	addMethod   func(cls, sel, imp uintptr, types string) bool
	allocPair   func(super uintptr, name string, extra uintptr) uintptr
	regPair     func(cls uintptr)

	msgU       func(id, sel uintptr) uintptr
	msgUU      func(id, sel, a uintptr) uintptr
	msgUUU     func(id, sel, a, b uintptr) uintptr
	msgUUUU    func(id, sel, a, b, c uintptr) uintptr
	msgU4      func(id, sel, a, b, c, d uintptr) uintptr
	msgStr     func(id, sel uintptr, p *byte) uintptr
	msgStrPtr  func(id, sel uintptr) *byte
	msgF       func(id, sel uintptr) float64                          // 返回 CGFloat
	msgRect    func(id, sel uintptr, r Rect) uintptr                  // 1 个 NSRect
	msgRect3   func(id, sel uintptr, r Rect, a, b, c uintptr) uintptr // NSRect + 3 个整数
	msgRectB   func(id, sel uintptr, r Rect, b uintptr) uintptr       // NSRect + BOOL
	msgGetRect func(id, sel uintptr) Rect                             // 返回 NSRect
	msgPtRect  func(id, sel uintptr, r Rect) Rect                     // NSRect → NSRect（转换）
	msgPoint   func(id, sel uintptr) Point                            // 返回 NSPoint
)

// Load 加载 libobjc + Foundation + AppKit 并解析符号（幂等）。
func Load() error {
	if loaded || loadErr != nil {
		return loadErr
	}
	h, err := purego.Dlopen("/usr/lib/libobjc.A.dylib", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		loadErr = errors.New("objc: 无法加载 libobjc：" + err.Error())
		return loadErr
	}
	_, _ = purego.Dlopen("/System/Library/Frameworks/Foundation.framework/Foundation", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	_, _ = purego.Dlopen("/System/Library/Frameworks/AppKit.framework/AppKit", purego.RTLD_NOW|purego.RTLD_GLOBAL)

	bind := func(dst any, name string) {
		if addr, e := purego.Dlsym(h, name); e == nil && addr != 0 {
			purego.RegisterFunc(dst, addr)
		}
	}
	bind(&objcGetCls, "objc_getClass")
	bind(&selRegName, "sel_registerName")
	bind(&addMethod, "class_addMethod")
	bind(&allocPair, "objc_allocateClassPair")
	bind(&regPair, "objc_registerClassPair")

	send, err := purego.Dlsym(h, "objc_msgSend")
	if err != nil || send == 0 {
		loadErr = errors.New("objc: libobjc 缺少 objc_msgSend")
		return loadErr
	}
	msgSendAddr = send
	purego.RegisterFunc(&msgU, send)
	purego.RegisterFunc(&msgUU, send)
	purego.RegisterFunc(&msgUUU, send)
	purego.RegisterFunc(&msgUUUU, send)
	purego.RegisterFunc(&msgU4, send)
	purego.RegisterFunc(&msgStr, send)
	purego.RegisterFunc(&msgStrPtr, send)
	purego.RegisterFunc(&msgF, send)
	purego.RegisterFunc(&msgRect, send)
	purego.RegisterFunc(&msgRect3, send)
	purego.RegisterFunc(&msgRectB, send)
	purego.RegisterFunc(&msgGetRect, send)
	purego.RegisterFunc(&msgPtRect, send)
	purego.RegisterFunc(&msgPoint, send)

	if objcGetCls == nil || selRegName == nil || msgU == nil {
		loadErr = errors.New("objc: libobjc 符号缺失")
		return loadErr
	}
	loaded = true
	return nil
}

// Class 按名字取类对象。
func Class(name string) uintptr { return objcGetCls(name) }

// Sel 注册（或查找）选择子。
func Sel(name string) uintptr { return selRegName(name) }

// Send 发消息（无参）。
func Send(id uintptr, sel string) uintptr { return msgU(id, Sel(sel)) }

// Send1 发消息（1 个参数）。
func Send1(id uintptr, sel string, a uintptr) uintptr { return msgUU(id, Sel(sel), a) }

// Send2 发消息（2 个参数）。
func Send2(id uintptr, sel string, a, b uintptr) uintptr { return msgUUU(id, Sel(sel), a, b) }

// Send3 发消息（3 个参数）。
func Send3(id uintptr, sel string, a, b, c uintptr) uintptr { return msgUUUU(id, Sel(sel), a, b, c) }

// Send4 发消息（4 个参数，如
// nextEventMatchingMask:untilDate:inMode:dequeue:）。
func Send4(id uintptr, sel string, a, b, c, d uintptr) uintptr {
	return msgU4(id, Sel(sel), a, b, c, d)
}

// FloatOf 取返回 CGFloat 的属性/方法（如 backingScaleFactor、scrollingDeltaY）。
func FloatOf(id uintptr, sel string) float64 { return msgF(id, Sel(sel)) }

// SendI 发消息（1 个 NSInteger 参数，如 setLevel:）。
func SendI(id uintptr, sel string, v int64) {
	purego.SyscallN(msgSendAddr, id, Sel(sel), uintptr(v))
}

// SendF 发消息（1 个 CGFloat 参数，如 setAlphaValue:）。
func SendF(id uintptr, sel string, v float64) {
	purego.SyscallN(msgSendAddr, id, Sel(sel), *(*uintptr)(unsafe.Pointer(&v)))
}

// SendB 发消息（1 个 BOOL 参数）。
func SendB(id uintptr, sel string, v bool) {
	var b uintptr
	if v {
		b = 1
	}
	purego.SyscallN(msgSendAddr, id, Sel(sel), b)
}

// SendRect 发消息（1 个 NSRect 参数，如 initWithFrame:）。
func SendRect(id uintptr, sel string, r Rect) uintptr { return msgRect(id, Sel(sel), r) }

// SendRect3 发消息（NSRect + 3 个整数参数，如
// initWithContentRect:styleMask:backing:defer:）。
func SendRect3(id uintptr, sel string, r Rect, a, b, c uintptr) uintptr {
	return msgRect3(id, Sel(sel), r, a, b, c)
}

// SendRectB 发消息（NSRect + BOOL，如 setFrame:display:）。
func SendRectB(id uintptr, sel string, r Rect, v bool) uintptr {
	var b uintptr
	if v {
		b = 1
	}
	return msgRectB(id, Sel(sel), r, b)
}

// RectOf 取返回 NSRect 的属性（如 bounds / frame）。
func RectOf(id uintptr, sel string) Rect { return msgGetRect(id, Sel(sel)) }

// RectConvert 取「NSRect 参数 → NSRect 返回」的方法（如 convertRectToBacking:）。
func RectConvert(id uintptr, sel string, r Rect) Rect { return msgPtRect(id, Sel(sel), r) }

// PointOf 取返回 NSPoint 的方法（如 locationInWindow）。
func PointOf(id uintptr, sel string) Point { return msgPoint(id, Sel(sel)) }

// BoolOf 取返回 BOOL 的方法（低 8 位）。
func BoolOf(id uintptr, sel string) bool { return byte(msgU(id, Sel(sel))) != 0 }

// IntOf 取返回 NSInteger 的方法。
func IntOf(id uintptr, sel string) int64 { return int64(msgU(id, Sel(sel))) }

// UTF8 NSString → Go 字符串（UTF8String + C 串读取）。
func UTF8(nsstr uintptr) string {
	if nsstr == 0 {
		return ""
	}
	// 用返回 *byte 的类型化签名，避免 uintptr → unsafe.Pointer 的转换
	return goString(msgStrPtr(nsstr, Sel("UTF8String")))
}

// NSString 由 Go 字符串创建临时 NSString。
func NSString(s string) uintptr {
	cls := Class("NSString")
	if cls == 0 {
		return 0
	}
	// 用带 *byte 的类型化签名：字符串缓冲在调用期间保持存活，
	// 比 uintptr(unsafe.Pointer(...)) 更安全（vet 也不会告警）
	return msgStr(cls, Sel("stringWithUTF8String:"), cstr(s))
}

// AddMethod 给类动态添加方法（imp 为 purego.NewCallback 得到的函数指针）。
// types 为 Objective-C 类型编码，例如 "v@:@" = void(id, SEL, id)。
func AddMethod(cls uintptr, name string, imp uintptr, types string) bool {
	if addMethod == nil || cls == 0 || imp == 0 {
		return false
	}
	return addMethod(cls, Sel(name), imp, types)
}

// Subclass 动态创建子类（已存在同名类时返回 0）。
func Subclass(super uintptr, name string) uintptr {
	if allocPair == nil || regPair == nil || super == 0 {
		return 0
	}
	cls := allocPair(super, name, 0)
	if cls != 0 {
		regPair(cls)
	}
	return cls
}

// U32Ptr 取 []uint32 的首元素指针（转 NSOpenGLPixelFormatAttribute*）。
func U32Ptr(v []uint32) uintptr {
	if len(v) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&v[0]))
}

// I32Ptr 取 *int32 的指针形式。
func I32Ptr(v *int32) uintptr { return uintptr(unsafe.Pointer(v)) }

// cstr 把 Go 字符串转成 NUL 结尾 C 串。
func cstr(s string) *byte {
	b := make([]byte, len(s)+1)
	copy(b, s)
	return &b[0]
}

// goString 读取 C 串。
func goString(p *byte) string {
	if p == nil {
		return ""
	}
	n := 0
	for *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + uintptr(n))) != 0 {
		n++
		if n > 1<<20 {
			break
		}
	}
	return string(unsafe.Slice(p, n))
}
