//go:build js && wasm

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package gpu

import (
	"fmt"
	"strings"
	"sync"
	"syscall/js"
	"unsafe"
)

// 浏览器端 GL 调用层：同一套渲染器代码跑在 WebGL 2 上。
//
// WebGL 的 shader / program / buffer / texture / framebuffer / VAO / uniform location
// 都是 JS 对象而不是整数句柄，而共享渲染器用 uint32/int32 保存它们，因此这里用一张
// 句柄表把 JS 对象映射成整数：返回值统一 reg() 登记，参数统一 obj() 取回。

var (
	glctx    js.Value // WebGL2RenderingContext
	glScr    js.Value // 复用的 Uint8Array，避免每次上传都新建
	glScrCap int

	objMu   sync.Mutex
	objTab  = map[uint32]js.Value{}
	objNext = uint32(1)
)

// InitWasm 用 canvas 创建 WebGL 2 上下文（必须在 Renderer.Init 之前调用）。
func InitWasm(canvas js.Value) error {
	// ?debug 打开 preserveDrawingBuffer，方便自动化冒烟测试直接 readPixels 取色。
	debug := strings.Contains(js.Global().Get("location").Get("search").String(), "debug")
	ctx := canvas.Call("getContext", "webgl2", map[string]any{
		"alpha":                 false,
		"antialias":             false,
		"premultipliedAlpha":    false,
		"preserveDrawingBuffer": debug,
		"desynchronized":        true,
	})
	if ctx.IsNull() || ctx.IsUndefined() {
		return fmt.Errorf("浏览器不支持 WebGL 2")
	}
	glctx = ctx
	return nil
}

// GLReady 上下文是否就绪。
func GLReady() bool { return glctx.Type() != js.TypeUndefined && glctx.Type() != js.TypeNull }

// reg 把 JS 对象登记成整数句柄。
func reg(v js.Value) uint32 {
	if v.IsNull() || v.IsUndefined() {
		return 0
	}
	objMu.Lock()
	defer objMu.Unlock()
	id := objNext
	objNext++
	objTab[id] = v
	return id
}

// obj 取回句柄对应的 JS 对象（未知句柄返回 null）。
func obj(id uint32) js.Value {
	if id == 0 {
		return js.Null()
	}
	objMu.Lock()
	defer objMu.Unlock()
	if v, ok := objTab[id]; ok {
		return v
	}
	return js.Null()
}

// drop 释放句柄。
func drop(id uint32) {
	if id == 0 {
		return
	}
	objMu.Lock()
	delete(objTab, id)
	objMu.Unlock()
}

// jsBytes 把 Go 字节切片复制到 JS 的 Uint8Array（复用同一块底层数组）。
func jsBytes(b []byte) js.Value {
	if len(b) == 0 {
		return js.Null()
	}
	if glScr.IsUndefined() || glScrCap < len(b) {
		n := len(b) * 2
		if n < 1<<16 {
			n = 1 << 16
		}
		glScr = js.Global().Get("Uint8Array").New(n)
		glScrCap = n
	}
	v := glScr.Call("subarray", 0, len(b))
	js.CopyBytesToJS(v, b)
	return v
}

// jsFloats 把 []float32 转成 JS 的 Float32Array。
func jsFloats(data []float32) js.Value {
	if len(data) == 0 {
		return js.Null()
	}
	raw := unsafe.Slice((*byte)(unsafe.Pointer(&data[0])), len(data)*4)
	u8 := jsBytes(raw)
	return js.Global().Get("Float32Array").New(u8.Get("buffer"), u8.Get("byteOffset"), len(data))
}

// toES3 把桌面版 GLSL 330 core 着色器翻译成 WebGL 2 需要的 GLSL ES 3.00。
func toES3(src string) string {
	if !strings.Contains(src, "#version 330 core") {
		return src
	}
	return strings.Replace(src, "#version 330 core",
		"#version 300 es\nprecision highp float;\nprecision highp int;", 1)
}

// jsInt 把 WebGL 的返回值统一成 int32：COMPILE_STATUS / LINK_STATUS 返回布尔，
// 其余参数（INFO_LOG_LENGTH 等）返回数字。
func jsInt(v js.Value) int32 {
	switch v.Type() {
	case js.TypeBoolean:
		if v.Bool() {
			return 1
		}
		return 0
	case js.TypeNumber:
		return int32(v.Int())
	}
	return 0
}

func glCall(name string, args ...any) js.Value { return glctx.Call(name, args...) }

func genIDs(kind string, n int32, ids *uint32) {
	for i := int32(0); i < n; i++ {
		*(*uint32)(unsafe.Add(unsafe.Pointer(ids), uintptr(i)*4)) = reg(glCall(kind))
	}
}

func delIDs(kind string, n int32, ids *uint32) {
	for i := int32(0); i < n; i++ {
		id := *(*uint32)(unsafe.Add(unsafe.Pointer(ids), uintptr(i)*4))
		if id != 0 {
			glCall(kind, obj(id))
			drop(id)
		}
	}
}

func glGenBuffers(n int32, ids *uint32) { genIDs("createBuffer", n, ids) }

func glBindBuffer(target uint32, id uint32) {
	if id == 0 {
		glCall("bindBuffer", int(target), js.Null())
		return
	}
	glCall("bindBuffer", int(target), obj(id))
}

func glBufferAlloc(target uint32, sizeBytes int, usage uint32) {
	glCall("bufferData", int(target), sizeBytes, int(usage))
}

func glBufferData(target uint32, data []float32, usage uint32) {
	glCall("bufferData", int(target), jsFloats(data), int(usage))
}

func glBufferSubData(target uint32, offset int, data []float32) {
	if len(data) == 0 {
		return
	}
	glCall("bufferSubData", int(target), offset, jsFloats(data))
}

func glDeleteBuffers(n int32, ids *uint32) { delIDs("deleteBuffer", n, ids) }

func glGenVertexArrays(n int32, ids *uint32) { genIDs("createVertexArray", n, ids) }

func glBindVertexArray(id uint32) {
	if id == 0 {
		glCall("bindVertexArray", js.Null())
		return
	}
	glCall("bindVertexArray", obj(id))
}

func glDeleteVertexArrays(n int32, ids *uint32) { delIDs("deleteVertexArray", n, ids) }

func glEnableVertexAttribArray(index uint32) { glCall("enableVertexAttribArray", int(index)) }

func glVertexAttribPointer(index uint32, size int32, typ uint32, normalized uint8, stride int32, offset int32) {
	glCall("vertexAttribPointer", int(index), int(size), int(typ), normalized != 0, int(stride), offset)
}

func glCreateShader(typ uint32) uint32 { return reg(glCall("createShader", int(typ))) }

func glShaderSource(shader uint32, src string) {
	glCall("shaderSource", obj(shader), toES3(src))
}

func glCompileShader(shader uint32) { glCall("compileShader", obj(shader)) }

func glGetShaderiv(shader uint32, pname uint32, params *int32) {
	*params = jsInt(glCall("getShaderParameter", obj(shader), int(pname)))
}

func glGetShaderInfoLog(shader uint32, maxLen int32, length *int32, info *byte) {
	s := glCall("getShaderInfoLog", obj(shader)).String()
	if len(s) > int(maxLen) {
		s = s[:maxLen]
	}
	copy(unsafe.Slice(info, maxLen), s)
	if length != nil {
		*length = int32(len(s))
	}
}

func glDeleteShader(shader uint32) {
	glCall("deleteShader", obj(shader))
	drop(shader)
}

func glCreateProgram() uint32 { return reg(glCall("createProgram")) }

func glAttachShader(program, shader uint32) { glCall("attachShader", obj(program), obj(shader)) }

func glLinkProgram(program uint32) { glCall("linkProgram", obj(program)) }

func glGetProgramiv(program uint32, pname uint32, params *int32) {
	*params = jsInt(glCall("getProgramParameter", obj(program), int(pname)))
}

func glGetProgramInfoLog(program uint32, maxLen int32, length *int32, info *byte) {
	s := glCall("getProgramInfoLog", obj(program)).String()
	if len(s) > int(maxLen) {
		s = s[:maxLen]
	}
	copy(unsafe.Slice(info, maxLen), s)
	if length != nil {
		*length = int32(len(s))
	}
}

func glUseProgram(program uint32) {
	if program == 0 {
		glCall("useProgram", js.Null())
		return
	}
	glCall("useProgram", obj(program))
}

func glDeleteProgram(program uint32) {
	glCall("deleteProgram", obj(program))
	drop(program)
}

func glGetUniformLocation(program uint32, name string) int32 {
	loc := glCall("getUniformLocation", obj(program), name)
	if loc.IsNull() || loc.IsUndefined() {
		return -1
	}
	return int32(reg(loc))
}

func glUniform1i(loc int32, v int32) {
	if loc >= 0 {
		glCall("uniform1i", obj(uint32(loc)), v)
	}
}

func glUniform1f(loc int32, v float32) {
	if loc >= 0 {
		glCall("uniform1f", obj(uint32(loc)), float64(v))
	}
}

func glUniform2f(loc int32, x, y float32) {
	if loc >= 0 {
		glCall("uniform2f", obj(uint32(loc)), float64(x), float64(y))
	}
}

func glGenTextures(n int32, ids *uint32) { genIDs("createTexture", n, ids) }

func glBindTexture(target uint32, id uint32) {
	if id == 0 {
		glCall("bindTexture", int(target), js.Null())
		return
	}
	glCall("bindTexture", int(target), obj(id))
}

func glDeleteTextures(n int32, ids *uint32) { delIDs("deleteTexture", n, ids) }

func glTexImage2D(target uint32, level, internalFormat int32, w, h int32, format, typ uint32, pixels []byte) {
	glCall("texImage2D", int(target), int(level), int(internalFormat),
		int(w), int(h), 0, int(format), int(typ), jsBytes(pixels))
}

func glTexSubImage2D(target uint32, level, x, y, w, h int32, format, typ uint32, pixels []byte) {
	if len(pixels) == 0 {
		return
	}
	glCall("texSubImage2D", int(target), int(level), int(x), int(y),
		int(w), int(h), int(format), int(typ), jsBytes(pixels))
}

func glTexParameteri(target, pname uint32, param int32) {
	glCall("texParameteri", int(target), int(pname), param)
}

func glActiveTexture(tex uint32) { glCall("activeTexture", int(tex)) }

func glPixelStorei(pname uint32, param int32) { glCall("pixelStorei", int(pname), param) }

func glGenFramebuffers(n int32, ids *uint32) { genIDs("createFramebuffer", n, ids) }

func glBindFramebuffer(target, fb uint32) {
	if fb == 0 {
		glCall("bindFramebuffer", int(target), js.Null())
		return
	}
	glCall("bindFramebuffer", int(target), obj(fb))
}

func glFramebufferTexture2D(target, attachment, textarget, texture uint32, level int32) {
	glCall("framebufferTexture2D", int(target), int(attachment), int(textarget), obj(texture), int(level))
}

func glCheckFramebufferStatus(target uint32) uint32 {
	return uint32(jsInt(glCall("checkFramebufferStatus", int(target))))
}

func glDeleteFramebuffers(n int32, ids *uint32) { delIDs("deleteFramebuffer", n, ids) }

func glCopyTexSubImage2D(target uint32, level, xoff, yoff, x, y, w, h int32) {
	glCall("copyTexSubImage2D", int(target), int(level), int(xoff), int(yoff), int(x), int(y), int(w), int(h))
}

func glViewport(x, y, w, h int32) { glCall("viewport", int(x), int(y), int(w), int(h)) }

// glGetString 浏览器端不直接暴露 GL 字符串，返回 nil（软渲染判定不适用）。
func glGetString(name uint32) *byte { return nil }

func glScissor(x, y, w, h int32) { glCall("scissor", int(x), int(y), int(w), int(h)) }
func glClearColor(r, g, b, a float32) {
	glCall("clearColor", float64(r), float64(g), float64(b), float64(a))
}
func glClear(mask uint32)       { glCall("clear", int(mask)) }
func glEnable(cap uint32)       { glCall("enable", int(cap)) }
func glDisable(cap uint32)      { glCall("disable", int(cap)) }
func glBlendFunc(sf, df uint32) { glCall("blendFunc", int(sf), int(df)) }

func glDrawArrays(mode uint32, first, count int32) {
	glCall("drawArrays", int(mode), int(first), int(count))
}
