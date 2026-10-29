package gpu

import "fmt"

func linkProgram(vs, fs string) (uint32, error) {
	v, err := compileShader(GL_VERTEX_SHADER, vs)
	if err != nil {
		return 0, err
	}
	f, err := compileShader(GL_FRAGMENT_SHADER, fs)
	if err != nil {
		glDeleteShader(v)
		return 0, err
	}
	p := glCreateProgram()
	glAttachShader(p, v)
	glAttachShader(p, f)
	glLinkProgram(p)

	var ok int32
	glGetProgramiv(p, GL_LINK_STATUS, &ok)
	glDeleteShader(v)
	glDeleteShader(f)
	if ok == 0 {
		log := shaderLog(p, true)
		glDeleteProgram(p)
		return 0, fmt.Errorf("link failed: %s", log)
	}
	return p, nil
}

func compileShader(typ uint32, src string) (uint32, error) {
	s := glCreateShader(typ)
	glShaderSource(s, src)
	glCompileShader(s)
	var ok int32
	glGetShaderiv(s, GL_COMPILE_STATUS, &ok)
	if ok == 0 {
		log := shaderLog(s, false)
		glDeleteShader(s)
		return 0, fmt.Errorf("compile failed: %s", log)
	}
	return s, nil
}

func shaderLog(obj uint32, isProgram bool) string {
	var n int32
	if isProgram {
		glGetProgramiv(obj, GL_INFO_LOG_LENGTH, &n)
	} else {
		glGetShaderiv(obj, GL_INFO_LOG_LENGTH, &n)
	}
	if n <= 1 {
		return "(no log)"
	}
	buf := make([]byte, n)
	var written int32
	if isProgram {
		glGetProgramInfoLog(obj, n, &written, &buf[0])
	} else {
		glGetShaderInfoLog(obj, n, &written, &buf[0])
	}
	if written > 0 && int(written) <= len(buf) {
		buf = buf[:written]
	}
	for len(buf) > 0 && buf[len(buf)-1] == 0 {
		buf = buf[:len(buf)-1]
	}
	return string(buf)
}
