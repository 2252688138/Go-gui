package gpu

import (
	"fmt"
	"unsafe"

	"gogui/internal/win32"
)

func linkProgram(vs, fs string) (uint32, error) {
	v, err := compileShader(win32.GL_VERTEX_SHADER, vs)
	if err != nil {
		return 0, err
	}
	f, err := compileShader(win32.GL_FRAGMENT_SHADER, fs)
	if err != nil {
		win32.GlDeleteShader(v)
		return 0, err
	}
	p := win32.GlCreateProgram()
	win32.GlAttachShader(p, v)
	win32.GlAttachShader(p, f)
	win32.GlLinkProgram(p)

	var ok int32
	win32.GlGetProgramiv(p, win32.GL_LINK_STATUS, &ok)
	win32.GlDeleteShader(v)
	win32.GlDeleteShader(f)
	if ok == 0 {
		log := shaderLog(p, true)
		win32.GlDeleteProgram(p)
		return 0, fmt.Errorf("link failed: %s", log)
	}
	return p, nil
}

func compileShader(typ uint32, src string) (uint32, error) {
	s := win32.GlCreateShader(typ)
	cs := cstr(src)
	l := int32(len(src))
	win32.GlShaderSource(s, 1, &cs, &l)
	win32.GlCompileShader(s)
	var ok int32
	win32.GlGetShaderiv(s, win32.GL_COMPILE_STATUS, &ok)
	if ok == 0 {
		log := shaderLog(s, false)
		win32.GlDeleteShader(s)
		return 0, fmt.Errorf("compile failed: %s", log)
	}
	return s, nil
}

func shaderLog(obj uint32, isProgram bool) string {
	var n int32
	if isProgram {
		win32.GlGetProgramiv(obj, win32.GL_INFO_LOG_LENGTH, &n)
	} else {
		win32.GlGetShaderiv(obj, win32.GL_INFO_LOG_LENGTH, &n)
	}
	if n <= 1 {
		return "(no log)"
	}
	buf := make([]byte, n)
	var written int32
	if isProgram {
		win32.GlGetProgramInfoLog(obj, n, &written, &buf[0])
	} else {
		win32.GlGetShaderInfoLog(obj, n, &written, &buf[0])
	}
	_ = unsafe.Pointer(nil)
	if written > 0 && int(written) <= len(buf) {
		buf = buf[:written]
	}
	for len(buf) > 0 && buf[len(buf)-1] == 0 {
		buf = buf[:len(buf)-1]
	}
	return string(buf)
}
