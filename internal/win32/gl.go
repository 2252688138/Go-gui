package win32

import (
	"github.com/ebitengine/purego"
)

// GL 常量（仅本框架用到的子集）
const (
	GL_TRIANGLES      = 0x0004
	GL_TRIANGLE_STRIP = 0x0005

	GL_SRC_ALPHA            = 0x0302
	GL_ONE_MINUS_SRC_ALPHA  = 0x0303
	GL_ONE                  = 1
	GL_ZERO                 = 0
	GL_SRC1_COLOR           = 0x88F9
	GL_ONE_MINUS_SRC1_COLOR = 0x88FA

	GL_DEPTH_TEST   = 0x0B71
	GL_BLEND        = 0x0BE2
	GL_SCISSOR_TEST = 0x0C11
	GL_CULL_FACE    = 0x0B44

	GL_COLOR_BUFFER_BIT = 0x00004000

	GL_ARRAY_BUFFER         = 0x8892
	GL_ELEMENT_ARRAY_BUFFER = 0x8893
	GL_STATIC_DRAW          = 0x88E4
	GL_DYNAMIC_DRAW         = 0x88E8
	GL_STREAM_DRAW          = 0x88E0

	GL_FLOAT         = 0x1406
	GL_UNSIGNED_INT  = 0x1405
	GL_UNSIGNED_BYTE = 0x1401

	GL_FRAGMENT_SHADER = 0x8B30
	GL_VERTEX_SHADER   = 0x8B31
	GL_COMPILE_STATUS  = 0x8B81
	GL_LINK_STATUS     = 0x8B82
	GL_INFO_LOG_LENGTH = 0x8B84

	GL_TEXTURE0 = 0x84C0
	GL_TEXTURE1 = 0x84C1

	GL_TEXTURE_2D                 = 0x0DE1
	GL_TEXTURE_MIN_FILTER         = 0x2801
	GL_TEXTURE_MAG_FILTER         = 0x2800
	GL_TEXTURE_WRAP_S             = 0x2802
	GL_TEXTURE_WRAP_T             = 0x2803
	GL_TEXTURE_WRAP_CLAMP         = 0x812F
	GL_TEXTURE_WRAP_CLAMP_TO_EDGE = 0x812F
	GL_LINEAR                     = 0x2601
	GL_NEAREST                    = 0x2600
	GL_RED                        = 0x1903
	GL_RGBA                       = 0x1908
	GL_RGBA8                      = 0x8058
	GL_R8                         = 0x8229
	GL_RED_INTEGER                = 0x8D94
	GL_UNPACK_ALIGNMENT           = 0x0CF5
	GL_PACK_ALIGNMENT             = 0x0D05

	GL_FRAMEBUFFER          = 0x8D40
	GL_RENDERBUFFER         = 0x8D41
	GL_COLOR_ATTACHMENT0    = 0x8CE0
	GL_FRAMEBUFFER_COMPLETE = 0x8CD5
	GL_TEXTURE_MAX_LEVEL    = 0x813D
	GL_FRAMEBUFFER_BINDING  = 0x8CA6
	GL_TEXTURE_BINDING_2D   = 0x8069
	GL_LINEAR_MIPMAP_LINEAR = 0x2703
	GL_CLAMP_TO_EDGE        = 0x812F

	GL_VENDOR   = 0x1F00
	GL_RENDERER = 0x1F01
	GL_VERSION  = 0x1F02
)

var (
	GlGenBuffers    func(n int32, ids *uint32)
	GlBindBuffer    func(target uint32, id uint32)
	GlBufferData    func(target uint32, size int, data unsafePointer, usage uint32)
	GlBufferSubData func(target uint32, offset int, size int, data unsafePointer)
	GlDeleteBuffers func(n int32, ids *uint32)

	GlGenVertexArrays         func(n int32, ids *uint32)
	GlBindVertexArray         func(id uint32)
	GlDeleteVertexArrays      func(n int32, ids *uint32)
	GlEnableVertexAttribArray func(index uint32)
	GlVertexAttribPointer     func(index uint32, size int32, typ uint32, normalized uint8, stride int32, ptr unsafePointer)

	GlCreateShader       func(typ uint32) uint32
	GlShaderSource       func(shader uint32, count int32, str **byte, length *int32)
	GlCompileShader      func(shader uint32)
	GlGetShaderiv        func(shader uint32, pname uint32, params *int32)
	GlGetShaderInfoLog   func(shader uint32, maxLen int32, length *int32, info *byte)
	GlDeleteShader       func(shader uint32)
	GlCreateProgram      func() uint32
	GlAttachShader       func(program, shader uint32)
	GlLinkProgram        func(program uint32)
	GlGetProgramiv       func(program uint32, pname uint32, params *int32)
	GlGetProgramInfoLog  func(program uint32, maxLen int32, length *int32, info *byte)
	GlUseProgram         func(program uint32)
	GlDeleteProgram      func(program uint32)
	GlGetUniformLocation func(program uint32, name *byte) int32
	GlUniform1i          func(loc int32, v int32)
	GlUniform1f          func(loc int32, v float32)
	GlUniform2f          func(loc int32, x, y float32)
	GlUniform3f          func(loc int32, x, y, z float32)
	GlUniform4f          func(loc int32, x, y, z, w float32)
	GlUniform4fv         func(loc int32, count int32, v *float32)
	GlUniformMatrix4fv   func(loc int32, count int32, transpose uint8, v *float32)

	GlGenTextures    func(n int32, ids *uint32)
	GlBindTexture    func(target uint32, id uint32)
	GlDeleteTextures func(n int32, ids *uint32)
	GlTexImage2D     func(target uint32, level int32, internalFormat int32, w, h int32, border int32, format, typ uint32, pixels unsafePointer)
	GlTexSubImage2D  func(target uint32, level int32, xoff, yoff, w, h int32, format, typ uint32, pixels unsafePointer)
	GlTexParameteri  func(target, pname uint32, param int32)
	GlActiveTexture  func(texture uint32)
	GlPixelStorei    func(pname uint32, param int32)

	GlGenFramebuffers        func(n int32, ids *uint32)
	GlBindFramebuffer        func(target, fb uint32)
	GlFramebufferTexture2D   func(target, attachment, textarget, texture uint32, level int32)
	GlCheckFramebufferStatus func(target uint32) uint32
	GlDeleteFramebuffers     func(n int32, ids *uint32)
	GlCopyTexSubImage2D      func(target uint32, level, xoff, yoff, x, y, w, h int32)
	GlTexImage2DMultisample  func(target uint32, samples int32, internalFormat int32, w, h int32, fixedsamplelocations uint8)
	GlViewport               func(x, y, w, h int32)
	GlScissor                func(x, y, w, h int32)
	GlClearColor             func(r, g, b, a float32)
	GlClear                  func(mask uint32)
	GlEnable                 func(cap uint32)
	GlDisable                func(cap uint32)
	GlBlendFunc              func(sf, df uint32)
	GlBlendFuncSeparate      func(sr, dr, sa, da uint32)
	GlDrawArrays             func(mode uint32, first, count int32)
	GlDrawElements           func(mode uint32, count int32, typ uint32, indices unsafePointer)
	GlGetString              func(name uint32) *byte
	GlGetError               func() uint32
	GlFinish                 func()
	GlFlush                  func()
	GlReadPixels             func(x, y, w, h int32, format, typ uint32, pixels unsafePointer)

	// VSync（WGL_EXT_swap_control 扩展，可选）
	wglSwapIntervalEXT func(interval int32) int32
)

func looksLikeGL1Name(name string) bool {
	return false
}

func resolveGL(name string) uintptr {
	if p := wglGetProcAddress(cstr(name)); p != 0 && p != 1 && p != 2 && p != 3 && p != ^uintptr(0) {
		return p
	}
	addr, err := procAddr(libOpengl32, name)
	if err != nil {
		return 0
	}
	return addr
}

func reg[T any](fn *T, name string) {
	addr := resolveGL(name)
	if addr == 0 {
		panic("opengl: missing symbol " + name)
	}
	purego.RegisterFunc(fn, addr)
}

func regOpt[T any](fn *T, name string) {
	addr := resolveGL(name)
	if addr == 0 {
		return
	}
	purego.RegisterFunc(fn, addr)
}

// InitGL 在所有 GL 函数指针就绪后调用（上下文已 current）。
func InitGL() {
	reg(&GlGenBuffers, "glGenBuffers")
	reg(&GlBindBuffer, "glBindBuffer")
	reg(&GlBufferData, "glBufferData")
	reg(&GlBufferSubData, "glBufferSubData")
	reg(&GlDeleteBuffers, "glDeleteBuffers")
	reg(&GlGenVertexArrays, "glGenVertexArrays")
	reg(&GlBindVertexArray, "glBindVertexArray")
	reg(&GlDeleteVertexArrays, "glDeleteVertexArrays")
	reg(&GlEnableVertexAttribArray, "glEnableVertexAttribArray")
	reg(&GlVertexAttribPointer, "glVertexAttribPointer")
	reg(&GlCreateShader, "glCreateShader")
	reg(&GlShaderSource, "glShaderSource")
	reg(&GlCompileShader, "glCompileShader")
	reg(&GlGetShaderiv, "glGetShaderiv")
	reg(&GlGetShaderInfoLog, "glGetShaderInfoLog")
	reg(&GlDeleteShader, "glDeleteShader")
	reg(&GlCreateProgram, "glCreateProgram")
	reg(&GlAttachShader, "glAttachShader")
	reg(&GlLinkProgram, "glLinkProgram")
	reg(&GlGetProgramiv, "glGetProgramiv")
	reg(&GlGetProgramInfoLog, "glGetProgramInfoLog")
	reg(&GlUseProgram, "glUseProgram")
	reg(&GlDeleteProgram, "glDeleteProgram")
	reg(&GlGetUniformLocation, "glGetUniformLocation")
	reg(&GlUniform1i, "glUniform1i")
	reg(&GlUniform1f, "glUniform1f")
	reg(&GlUniform2f, "glUniform2f")
	reg(&GlUniform3f, "glUniform3f")
	reg(&GlUniform4f, "glUniform4f")
	reg(&GlUniform4fv, "glUniform4fv")
	reg(&GlUniformMatrix4fv, "glUniformMatrix4fv")
	reg(&GlGenTextures, "glGenTextures")
	reg(&GlBindTexture, "glBindTexture")
	reg(&GlDeleteTextures, "glDeleteTextures")
	reg(&GlTexImage2D, "glTexImage2D")
	reg(&GlTexSubImage2D, "glTexSubImage2D")
	reg(&GlTexParameteri, "glTexParameteri")
	reg(&GlActiveTexture, "glActiveTexture")
	reg(&GlPixelStorei, "glPixelStorei")
	reg(&GlGenFramebuffers, "glGenFramebuffers")
	reg(&GlBindFramebuffer, "glBindFramebuffer")
	reg(&GlFramebufferTexture2D, "glFramebufferTexture2D")
	reg(&GlCheckFramebufferStatus, "glCheckFramebufferStatus")
	reg(&GlDeleteFramebuffers, "glDeleteFramebuffers")
	reg(&GlCopyTexSubImage2D, "glCopyTexSubImage2D")
	regOpt(&GlTexImage2DMultisample, "glTexImage2DMultisample")
	reg(&GlViewport, "glViewport")
	reg(&GlScissor, "glScissor")
	reg(&GlClearColor, "glClearColor")
	reg(&GlClear, "glClear")
	reg(&GlEnable, "glEnable")
	reg(&GlDisable, "glDisable")
	reg(&GlBlendFunc, "glBlendFunc")
	regOpt(&GlBlendFuncSeparate, "glBlendFuncSeparate")
	reg(&GlDrawArrays, "glDrawArrays")
	reg(&GlDrawElements, "glDrawElements")
	reg(&GlGetString, "glGetString")
	reg(&GlGetError, "glGetError")
	reg(&GlFinish, "glFinish")
	reg(&GlFlush, "glFlush")
	reg(&GlReadPixels, "glReadPixels")
	// 垂直同步：避免帧与显示器刷新错位产生撕裂/抖动（扩展不存在时忽略）
	regOpt(&wglSwapIntervalEXT, "wglSwapIntervalEXT")
}
