package gpu

// OpenGL 3.3 / WebGL 2 共用的常量子集。
//
// 两个 API 的枚举值一致，因此常量只在这里定义一份；具体调用由 glport_desktop.go
// （Win32 + wgl，函数指针经 purego 解析）与 glport_wasm.go（WebGL 2，syscall/js）
// 分别实现，渲染器本身不再认识任何平台细节。
const (
	GL_TRIANGLES      = 0x0004
	GL_TRIANGLE_STRIP = 0x0005

	GL_SRC_ALPHA           = 0x0302
	GL_ONE_MINUS_SRC_ALPHA = 0x0303
	GL_ONE                 = 1
	GL_ZERO                = 0

	GL_DEPTH_TEST   = 0x0B71
	GL_BLEND        = 0x0BE2
	GL_SCISSOR_TEST = 0x0C11
	GL_CULL_FACE    = 0x0B44

	GL_COLOR_BUFFER_BIT = 0x00004000

	GL_ARRAY_BUFFER = 0x8892
	GL_STATIC_DRAW  = 0x88E4
	GL_STREAM_DRAW  = 0x88E0

	GL_FLOAT         = 0x1406
	GL_UNSIGNED_INT  = 0x1405
	GL_UNSIGNED_BYTE = 0x1401

	GL_FRAGMENT_SHADER = 0x8B30
	GL_VERTEX_SHADER   = 0x8B31
	GL_COMPILE_STATUS  = 0x8B81
	GL_LINK_STATUS     = 0x8B82
	GL_INFO_LOG_LENGTH = 0x8B84

	GL_TEXTURE0 = 0x84C0

	GL_TEXTURE_2D                 = 0x0DE1
	GL_TEXTURE_MIN_FILTER         = 0x2801
	GL_TEXTURE_MAG_FILTER         = 0x2800
	GL_TEXTURE_WRAP_S             = 0x2802
	GL_TEXTURE_WRAP_T             = 0x2803
	GL_TEXTURE_WRAP_CLAMP_TO_EDGE = 0x812F
	GL_LINEAR                     = 0x2601
	GL_RED                        = 0x1903
	GL_RGBA                       = 0x1908
	GL_RGBA8                      = 0x8058
	GL_R8                         = 0x8229
	GL_UNPACK_ALIGNMENT           = 0x0CF5

	GL_FRAMEBUFFER          = 0x8D40
	GL_COLOR_ATTACHMENT0    = 0x8CE0
	GL_FRAMEBUFFER_COMPLETE = 0x8CD5
)
