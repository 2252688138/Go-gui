//go:build linux || darwin || freebsd || android

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package gpu

import (
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// 类 Unix 端（Linux / macOS / BSD / Android）GL 调用层：零 cgo。
//
// 与 Windows 端 glport_win32.go（转发 win32.Gl*）对应，这里用 purego 动态加载
// libGL / libGLESv3 并解析函数指针：
//
//  1. 后端（internal/x11 的 EGL/GLX、internal/cocoa 的 NSOpenGL、Android 的 EGL）
//     创建上下文后调用 SetGLProcLoader，把「按名字查函数地址」的能力交给本包；
//  2. InitGLProcs 解析整张函数表（缺失符号保持 nil，调用时安全空操作）；
//  3. 渲染器只认识 glXxx 这组薄包装，因此与平台完全解耦。
//
// 空操作而非 panic 是有意为之：未初始化时窗口后端会先返回错误，
// 万一仍有调用也不会把进程打死（与「不支持的能力空实现」约定一致）。

// glProc 按名字返回 GL 函数地址（eglGetProcAddress / glXGetProcAddress / dlsym 均可）。
type glProc func(name string) uintptr

var (
	glMu     sync.RWMutex
	glProcFn glProc
	glTable  glFns
	glReady  bool
)

// glFns 桌面 GL / GLES 函数表（名称与 OpenGL 3.3 一致）。
type glFns struct {
	GenBuffers              func(int32, *uint32)
	BindBuffer              func(uint32, uint32)
	BufferData              func(uint32, int, unsafe.Pointer, uint32)
	BufferSubData           func(uint32, int, int, unsafe.Pointer)
	DeleteBuffers           func(int32, *uint32)
	GenVertexArrays         func(int32, *uint32)
	BindVertexArray         func(uint32)
	DeleteVertexArrays      func(int32, *uint32)
	EnableVertexAttribArray func(uint32)
	VertexAttribPointer     func(uint32, int32, uint32, uint8, int32, unsafe.Pointer)
	CreateShader            func(uint32) uint32
	ShaderSource            func(uint32, int32, **byte, *int32)
	CompileShader           func(uint32)
	GetShaderiv             func(uint32, uint32, *int32)
	GetShaderInfoLog        func(uint32, int32, *int32, *byte)
	DeleteShader            func(uint32)
	CreateProgram           func() uint32
	AttachShader            func(uint32, uint32)
	LinkProgram             func(uint32)
	GetProgramiv            func(uint32, uint32, *int32)
	GetProgramInfoLog       func(uint32, int32, *int32, *byte)
	UseProgram              func(uint32)
	DeleteProgram           func(uint32)
	GetUniformLocation      func(uint32, *byte) int32
	Uniform1i               func(int32, int32)
	Uniform1f               func(int32, float32)
	Uniform2f               func(int32, float32, float32)
	GenTextures             func(int32, *uint32)
	BindTexture             func(uint32, uint32)
	DeleteTextures          func(int32, *uint32)
	TexImage2D              func(uint32, int32, int32, int32, int32, int32, uint32, uint32, unsafe.Pointer)
	TexSubImage2D           func(uint32, int32, int32, int32, int32, int32, uint32, uint32, unsafe.Pointer)
	TexParameteri           func(uint32, uint32, int32)
	ActiveTexture           func(uint32)
	PixelStorei             func(uint32, int32)
	GenFramebuffers         func(int32, *uint32)
	BindFramebuffer         func(uint32, uint32)
	FramebufferTexture2D    func(uint32, uint32, uint32, uint32, int32)
	CheckFramebufferStatus  func(uint32) uint32
	DeleteFramebuffers      func(int32, *uint32)
	CopyTexSubImage2D       func(uint32, int32, int32, int32, int32, int32, int32, int32)
	GetString               func(uint32) *byte
	Viewport                func(int32, int32, int32, int32)
	Scissor                 func(int32, int32, int32, int32)
	ClearColor              func(float32, float32, float32, float32)
	Clear                   func(uint32)
	Enable                  func(uint32)
	Disable                 func(uint32)
	BlendFunc               func(uint32, uint32)
	DrawArrays              func(uint32, int32, int32)
}

// SetGLProcLoader 由窗口后端注入「按名字查函数地址」的实现（须在 InitGLProcs 之前）。
func SetGLProcLoader(fn glProc) {
	glMu.Lock()
	glProcFn = fn
	glReady = false
	glMu.Unlock()
}

// GLReady 是否已成功解析 GL 函数表。
func GLReady() bool {
	glMu.RLock()
	ok := glReady
	glMu.RUnlock()
	return ok
}

// InitGLProcs 解析整张 GL 函数表。
//
// 缺符号不报错（保持 nil → 调用时空操作），但返回解析到的函数个数，
// 供后端判断上下文是否真的可用（正常情况应为 len(table)）。
func InitGLProcs() int {
	glMu.Lock()
	defer glMu.Unlock()
	if glProcFn == nil {
		return 0
	}
	t := glFns{}
	n := 0
	bind := func(dst any, name string) {
		if addr := glProcFn(name); addr != 0 {
			purego.RegisterFunc(dst, addr)
			n++
		}
	}
	bind(&t.GenBuffers, "glGenBuffers")
	bind(&t.BindBuffer, "glBindBuffer")
	bind(&t.BufferData, "glBufferData")
	bind(&t.BufferSubData, "glBufferSubData")
	bind(&t.DeleteBuffers, "glDeleteBuffers")
	bind(&t.GenVertexArrays, "glGenVertexArrays")
	bind(&t.BindVertexArray, "glBindVertexArray")
	bind(&t.DeleteVertexArrays, "glDeleteVertexArrays")
	bind(&t.EnableVertexAttribArray, "glEnableVertexAttribArray")
	bind(&t.VertexAttribPointer, "glVertexAttribPointer")
	bind(&t.CreateShader, "glCreateShader")
	bind(&t.ShaderSource, "glShaderSource")
	bind(&t.CompileShader, "glCompileShader")
	bind(&t.GetShaderiv, "glGetShaderiv")
	bind(&t.GetShaderInfoLog, "glGetShaderInfoLog")
	bind(&t.DeleteShader, "glDeleteShader")
	bind(&t.CreateProgram, "glCreateProgram")
	bind(&t.AttachShader, "glAttachShader")
	bind(&t.LinkProgram, "glLinkProgram")
	bind(&t.GetProgramiv, "glGetProgramiv")
	bind(&t.GetProgramInfoLog, "glGetProgramInfoLog")
	bind(&t.UseProgram, "glUseProgram")
	bind(&t.DeleteProgram, "glDeleteProgram")
	bind(&t.GetUniformLocation, "glGetUniformLocation")
	bind(&t.Uniform1i, "glUniform1i")
	bind(&t.Uniform1f, "glUniform1f")
	bind(&t.Uniform2f, "glUniform2f")
	bind(&t.GenTextures, "glGenTextures")
	bind(&t.BindTexture, "glBindTexture")
	bind(&t.DeleteTextures, "glDeleteTextures")
	bind(&t.TexImage2D, "glTexImage2D")
	bind(&t.TexSubImage2D, "glTexSubImage2D")
	bind(&t.TexParameteri, "glTexParameteri")
	bind(&t.ActiveTexture, "glActiveTexture")
	bind(&t.PixelStorei, "glPixelStorei")
	bind(&t.GenFramebuffers, "glGenFramebuffers")
	bind(&t.BindFramebuffer, "glBindFramebuffer")
	bind(&t.FramebufferTexture2D, "glFramebufferTexture2D")
	bind(&t.CheckFramebufferStatus, "glCheckFramebufferStatus")
	bind(&t.DeleteFramebuffers, "glDeleteFramebuffers")
	bind(&t.CopyTexSubImage2D, "glCopyTexSubImage2D")
	bind(&t.GetString, "glGetString")
	bind(&t.Viewport, "glViewport")
	bind(&t.Scissor, "glScissor")
	bind(&t.ClearColor, "glClearColor")
	bind(&t.Clear, "glClear")
	bind(&t.Enable, "glEnable")
	bind(&t.Disable, "glDisable")
	bind(&t.BlendFunc, "glBlendFunc")
	bind(&t.DrawArrays, "glDrawArrays")

	glTable = t
	glReady = n > 0
	return n
}

// fns 取函数表快照（nil 字段由各包装函数空值保护）。
func fns() *glFns {
	glMu.RLock()
	t := &glTable
	glMu.RUnlock()
	return t
}

// cstr 把 Go 字符串转成 GL 需要的 NUL 结尾 C 串（仅桌面端需要）。
func cstr(s string) *byte {
	b := make([]byte, len(s)+1)
	copy(b, s)
	return &b[0]
}

// ---- 51 个薄包装（与 glport_win32.go 逐一对应）----

func glGenBuffers(n int32, ids *uint32) {
	if f := fns().GenBuffers; f != nil {
		f(n, ids)
	}
}

func glBindBuffer(target uint32, id uint32) {
	if f := fns().BindBuffer; f != nil {
		f(target, id)
	}
}

func glBufferAlloc(target uint32, sizeBytes int, usage uint32) {
	if f := fns().BufferData; f != nil {
		f(target, sizeBytes, nil, usage)
	}
}

func glBufferData(target uint32, data []float32, usage uint32) {
	f := fns().BufferData
	if f == nil {
		return
	}
	var p unsafe.Pointer
	if len(data) > 0 {
		p = unsafe.Pointer(&data[0])
	}
	f(target, len(data)*4, p, usage)
}

func glBufferSubData(target uint32, offset int, data []float32) {
	f := fns().BufferSubData
	if f == nil || len(data) == 0 {
		return
	}
	f(target, offset, len(data)*4, unsafe.Pointer(&data[0]))
}

func glDeleteBuffers(n int32, ids *uint32) {
	if f := fns().DeleteBuffers; f != nil {
		f(n, ids)
	}
}

func glGenVertexArrays(n int32, ids *uint32) {
	if f := fns().GenVertexArrays; f != nil {
		f(n, ids)
	}
}

func glBindVertexArray(id uint32) {
	if f := fns().BindVertexArray; f != nil {
		f(id)
	}
}

func glDeleteVertexArrays(n int32, ids *uint32) {
	if f := fns().DeleteVertexArrays; f != nil {
		f(n, ids)
	}
}

func glEnableVertexAttribArray(index uint32) {
	if f := fns().EnableVertexAttribArray; f != nil {
		f(index)
	}
}

func glVertexAttribPointer(index uint32, size int32, typ uint32, normalized uint8, stride int32, offset int32) {
	if f := fns().VertexAttribPointer; f != nil {
		f(index, size, typ, normalized, stride, unsafe.Pointer(uintptr(offset)))
	}
}

func glCreateShader(typ uint32) uint32 {
	if f := fns().CreateShader; f != nil {
		return f(typ)
	}
	return 0
}

func glShaderSource(shader uint32, src string) {
	f := fns().ShaderSource
	if f == nil {
		return
	}
	cs := cstr(src)
	l := int32(len(src))
	f(shader, 1, &cs, &l)
}

func glCompileShader(shader uint32) {
	if f := fns().CompileShader; f != nil {
		f(shader)
	}
}

func glGetShaderiv(shader uint32, pname uint32, params *int32) {
	if f := fns().GetShaderiv; f != nil {
		f(shader, pname, params)
	}
}

func glGetShaderInfoLog(shader uint32, maxLen int32, length *int32, info *byte) {
	if f := fns().GetShaderInfoLog; f != nil {
		f(shader, maxLen, length, info)
	}
}

func glDeleteShader(shader uint32) {
	if f := fns().DeleteShader; f != nil {
		f(shader)
	}
}

func glCreateProgram() uint32 {
	if f := fns().CreateProgram; f != nil {
		return f()
	}
	return 0
}

func glAttachShader(program, shader uint32) {
	if f := fns().AttachShader; f != nil {
		f(program, shader)
	}
}

func glLinkProgram(program uint32) {
	if f := fns().LinkProgram; f != nil {
		f(program)
	}
}

func glGetProgramiv(program uint32, pname uint32, params *int32) {
	if f := fns().GetProgramiv; f != nil {
		f(program, pname, params)
	}
}

func glGetProgramInfoLog(program uint32, maxLen int32, length *int32, info *byte) {
	if f := fns().GetProgramInfoLog; f != nil {
		f(program, maxLen, length, info)
	}
}

func glUseProgram(program uint32) {
	if f := fns().UseProgram; f != nil {
		f(program)
	}
}

func glDeleteProgram(program uint32) {
	if f := fns().DeleteProgram; f != nil {
		f(program)
	}
}

func glGetUniformLocation(program uint32, name string) int32 {
	if f := fns().GetUniformLocation; f != nil {
		return f(program, cstr(name))
	}
	return -1
}

func glUniform1i(loc int32, v int32) {
	if f := fns().Uniform1i; f != nil {
		f(loc, v)
	}
}

func glUniform1f(loc int32, v float32) {
	if f := fns().Uniform1f; f != nil {
		f(loc, v)
	}
}

func glUniform2f(loc int32, x, y float32) {
	if f := fns().Uniform2f; f != nil {
		f(loc, x, y)
	}
}

func glGenTextures(n int32, ids *uint32) {
	if f := fns().GenTextures; f != nil {
		f(n, ids)
	}
}

func glBindTexture(target uint32, id uint32) {
	if f := fns().BindTexture; f != nil {
		f(target, id)
	}
}

func glDeleteTextures(n int32, ids *uint32) {
	if f := fns().DeleteTextures; f != nil {
		f(n, ids)
	}
}

func glTexImage2D(target uint32, level, internalFormat int32, w, h int32, format, typ uint32, pixels []byte) {
	f := fns().TexImage2D
	if f == nil {
		return
	}
	var p unsafe.Pointer
	if len(pixels) > 0 {
		p = unsafe.Pointer(&pixels[0])
	}
	f(target, level, internalFormat, w, h, 0, format, typ, p)
}

func glTexSubImage2D(target uint32, level, x, y, w, h int32, format, typ uint32, pixels []byte) {
	f := fns().TexSubImage2D
	if f == nil || len(pixels) == 0 {
		return
	}
	f(target, level, x, y, w, h, format, typ, unsafe.Pointer(&pixels[0]))
}

func glTexParameteri(target, pname uint32, param int32) {
	if f := fns().TexParameteri; f != nil {
		f(target, pname, param)
	}
}

func glActiveTexture(tex uint32) {
	if f := fns().ActiveTexture; f != nil {
		f(tex)
	}
}

func glPixelStorei(pname uint32, param int32) {
	if f := fns().PixelStorei; f != nil {
		f(pname, param)
	}
}

func glGenFramebuffers(n int32, ids *uint32) {
	if f := fns().GenFramebuffers; f != nil {
		f(n, ids)
	}
}

func glBindFramebuffer(target, fb uint32) {
	if f := fns().BindFramebuffer; f != nil {
		f(target, fb)
	}
}

func glFramebufferTexture2D(target, attachment, textarget, texture uint32, level int32) {
	if f := fns().FramebufferTexture2D; f != nil {
		f(target, attachment, textarget, texture, level)
	}
}

func glCheckFramebufferStatus(target uint32) uint32 {
	if f := fns().CheckFramebufferStatus; f != nil {
		return f(target)
	}
	return 0
}

func glDeleteFramebuffers(n int32, ids *uint32) {
	if f := fns().DeleteFramebuffers; f != nil {
		f(n, ids)
	}
}

func glCopyTexSubImage2D(target uint32, level, xoff, yoff, x, y, w, h int32) {
	if f := fns().CopyTexSubImage2D; f != nil {
		f(target, level, xoff, yoff, x, y, w, h)
	}
}

func glViewport(x, y, w, h int32) {
	if f := fns().Viewport; f != nil {
		f(x, y, w, h)
	}
}

// glGetString 读取 GL 字符串（GL_RENDERER 用于识别软件光栅）。
func glGetString(name uint32) *byte {
	if f := fns().GetString; f != nil {
		return f(name)
	}
	return nil
}

func glScissor(x, y, w, h int32) {
	if f := fns().Scissor; f != nil {
		f(x, y, w, h)
	}
}

func glClearColor(r, g, b, a float32) {
	if f := fns().ClearColor; f != nil {
		f(r, g, b, a)
	}
}

func glClear(mask uint32) {
	if f := fns().Clear; f != nil {
		f(mask)
	}
}

func glEnable(cap uint32) {
	if f := fns().Enable; f != nil {
		f(cap)
	}
}

func glDisable(cap uint32) {
	if f := fns().Disable; f != nil {
		f(cap)
	}
}

func glBlendFunc(sf, df uint32) {
	if f := fns().BlendFunc; f != nil {
		f(sf, df)
	}
}

func glDrawArrays(mode uint32, first, count int32) {
	if f := fns().DrawArrays; f != nil {
		f(mode, first, count)
	}
}
