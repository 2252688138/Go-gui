// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package gpu

import "unsafe"

// RendererName 当前 GL 渲染器名称（glGetString(GL_RENDERER)），失败返回空串。
//
// 用途：识别软件光栅（llvmpipe / softpipe / swrast / Microsoft Basic Render 等）——
// 这类后端每一帧都是用 CPU 填充像素，全屏 + 整帧超采样会慢到看起来像「黑屏」。
func RendererName() string {
	p := glGetString(GL_RENDERER)
	if p == nil {
		return ""
	}
	// glGetString 返回以 NUL 结尾的 C 字符串，最长按 256 字节截断
	var b []byte
	for i := uintptr(0); i < 256; i++ {
		c := *(*byte)(unsafe.Add(unsafe.Pointer(p), i))
		if c == 0 {
			break
		}
		b = append(b, c)
	}
	return string(b)
}

// SoftwareRenderer 当前是否为软件渲染后端。
func SoftwareRenderer() bool {
	switch name := lowerASCII(RendererName()); {
	case containsASCII(name, "llvmpipe"),
		containsASCII(name, "softpipe"),
		containsASCII(name, "swrast"),
		containsASCII(name, "software"),
		containsASCII(name, "basic render"):
		return true
	}
	return false
}

// lowerASCII 只处理 ASCII 大小写（渲染器名恒为 ASCII，避免引入 strings 依赖）。
func lowerASCII(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}

func containsASCII(s, sub string) bool {
	if len(sub) == 0 || len(sub) > len(s) {
		return false
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
