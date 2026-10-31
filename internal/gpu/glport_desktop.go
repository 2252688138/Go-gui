//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package gpu

import (
	"unsafe"

	"gogui/internal/win32"
)

// 桌面端 GL 调用层：薄薄一层包装，把「切片/字符串」形式的参数翻译成
// wgl 需要的裸指针，渲染器因此可以同时在桌面与 WebGL 2（切片 API）上编译。

func glGenBuffers(n int32, ids *uint32) { win32.GlGenBuffers(n, ids) }
func glBindBuffer(target uint32, id uint32) {
	win32.GlBindBuffer(target, id)
}
func glBufferAlloc(target uint32, sizeBytes int, usage uint32) {
	win32.GlBufferData(target, sizeBytes, nil, usage)
}
func glBufferData(target uint32, data []float32, usage uint32) {
	var p unsafe.Pointer
	if len(data) > 0 {
		p = unsafe.Pointer(&data[0])
	}
	win32.GlBufferData(target, len(data)*4, p, usage)
}
func glBufferSubData(target uint32, offset int, data []float32) {
	if len(data) == 0 {
		return
	}
	win32.GlBufferSubData(target, offset, len(data)*4, unsafe.Pointer(&data[0]))
}
func glDeleteBuffers(n int32, ids *uint32) { win32.GlDeleteBuffers(n, ids) }

func glGenVertexArrays(n int32, ids *uint32)    { win32.GlGenVertexArrays(n, ids) }
func glBindVertexArray(id uint32)               { win32.GlBindVertexArray(id) }
func glDeleteVertexArrays(n int32, ids *uint32) { win32.GlDeleteVertexArrays(n, ids) }
func glEnableVertexAttribArray(index uint32)    { win32.GlEnableVertexAttribArray(index) }
func glVertexAttribPointer(index uint32, size int32, typ uint32, normalized uint8, stride int32, offset int32) {
	win32.GlVertexAttribPointer(index, size, typ, normalized, stride, unsafe.Pointer(uintptr(offset)))
}

func glCreateShader(typ uint32) uint32 { return win32.GlCreateShader(typ) }

// cstr 把 Go 字符串转成 GL 需要的 NUL 结尾 C 串（仅桌面端需要）。
func cstr(s string) *byte {
	b := make([]byte, len(s)+1)
	copy(b, s)
	return &b[0]
}
func glShaderSource(shader uint32, src string) {
	cs := cstr(src)
	l := int32(len(src))
	win32.GlShaderSource(shader, 1, &cs, &l)
}
func glCompileShader(shader uint32) { win32.GlCompileShader(shader) }
func glGetShaderiv(shader uint32, pname uint32, params *int32) {
	win32.GlGetShaderiv(shader, pname, params)
}
func glGetShaderInfoLog(shader uint32, maxLen int32, length *int32, info *byte) {
	win32.GlGetShaderInfoLog(shader, maxLen, length, info)
}
func glDeleteShader(shader uint32) { win32.GlDeleteShader(shader) }

func glCreateProgram() uint32               { return win32.GlCreateProgram() }
func glAttachShader(program, shader uint32) { win32.GlAttachShader(program, shader) }
func glLinkProgram(program uint32)          { win32.GlLinkProgram(program) }
func glGetProgramiv(program uint32, pname uint32, params *int32) {
	win32.GlGetProgramiv(program, pname, params)
}
func glGetProgramInfoLog(program uint32, maxLen int32, length *int32, info *byte) {
	win32.GlGetProgramInfoLog(program, maxLen, length, info)
}
func glUseProgram(program uint32)    { win32.GlUseProgram(program) }
func glDeleteProgram(program uint32) { win32.GlDeleteProgram(program) }
func glGetUniformLocation(program uint32, name string) int32 {
	return win32.GlGetUniformLocation(program, cstr(name))
}
func glUniform1i(loc int32, v int32)      { win32.GlUniform1i(loc, v) }
func glUniform1f(loc int32, v float32)    { win32.GlUniform1f(loc, v) }
func glUniform2f(loc int32, x, y float32) { win32.GlUniform2f(loc, x, y) }

func glGenTextures(n int32, ids *uint32)     { win32.GlGenTextures(n, ids) }
func glBindTexture(target uint32, id uint32) { win32.GlBindTexture(target, id) }
func glDeleteTextures(n int32, ids *uint32)  { win32.GlDeleteTextures(n, ids) }
func glTexImage2D(target uint32, level, internalFormat int32, w, h int32, format, typ uint32, pixels []byte) {
	var p unsafe.Pointer
	if len(pixels) > 0 {
		p = unsafe.Pointer(&pixels[0])
	}
	win32.GlTexImage2D(target, level, internalFormat, w, h, 0, format, typ, p)
}
func glTexSubImage2D(target uint32, level, x, y, w, h int32, format, typ uint32, pixels []byte) {
	if len(pixels) == 0 {
		return
	}
	win32.GlTexSubImage2D(target, level, x, y, w, h, format, typ, unsafe.Pointer(&pixels[0]))
}
func glTexParameteri(target, pname uint32, param int32) {
	win32.GlTexParameteri(target, pname, param)
}
func glActiveTexture(tex uint32)              { win32.GlActiveTexture(tex) }
func glPixelStorei(pname uint32, param int32) { win32.GlPixelStorei(pname, param) }

func glGenFramebuffers(n int32, ids *uint32) { win32.GlGenFramebuffers(n, ids) }
func glBindFramebuffer(target, fb uint32)    { win32.GlBindFramebuffer(target, fb) }
func glFramebufferTexture2D(target, attachment, textarget, texture uint32, level int32) {
	win32.GlFramebufferTexture2D(target, attachment, textarget, texture, level)
}
func glCheckFramebufferStatus(target uint32) uint32 {
	return win32.GlCheckFramebufferStatus(target)
}
func glDeleteFramebuffers(n int32, ids *uint32) { win32.GlDeleteFramebuffers(n, ids) }
func glCopyTexSubImage2D(target uint32, level, xoff, yoff, x, y, w, h int32) {
	win32.GlCopyTexSubImage2D(target, level, xoff, yoff, x, y, w, h)
}

func glViewport(x, y, w, h int32) { win32.GlViewport(x, y, w, h) }

// glGetString 读取 GL 字符串（GL_RENDERER 用于识别软件光栅）。
// 未初始化 GL 时 win32.GlGetString 为 nil，必须判空（否则单元测试里直接 panic）。
func glGetString(name uint32) *byte {
	if win32.GlGetString == nil {
		return nil
	}
	return win32.GlGetString(name)
}

func glScissor(x, y, w, h int32)                   { win32.GlScissor(x, y, w, h) }
func glClearColor(r, g, b, a float32)              { win32.GlClearColor(r, g, b, a) }
func glClear(mask uint32)                          { win32.GlClear(mask) }
func glEnable(cap uint32)                          { win32.GlEnable(cap) }
func glDisable(cap uint32)                         { win32.GlDisable(cap) }
func glBlendFunc(sf, df uint32)                    { win32.GlBlendFunc(sf, df) }
func glDrawArrays(mode uint32, first, count int32) { win32.GlDrawArrays(mode, first, count) }
