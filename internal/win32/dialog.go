//go:build windows

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package win32

import (
	"strings"
	"syscall"
	"unsafe"
)

// FileDialogFilter 把 "文本 (*.txt)|*.txt|所有文件|*.*" 转成 Win32 双 0 结尾格式。
func FileDialogFilter(spec string) []uint16 {
	if spec == "" {
		spec = "所有文件 (*.*)|*.*"
	}
	var out []uint16
	for _, part := range strings.Split(spec, "|") {
		out = append(out, syscall.StringToUTF16(part)...)
	}
	out = append(out, 0) // 末尾双 0
	return out
}

func u16p(s []uint16) *uint16 {
	if len(s) == 0 {
		return nil
	}
	return &s[0]
}

// openFileDialog 通用打开/保存对话框。
func openFileDialog(owner uintptr, title, filter, initialDir, defExt string, save bool) (string, bool) {
	fileBuf := make([]uint16, 4096)
	ft := FileDialogFilter(filter)
	var titleP, dirP, extP *uint16
	if title != "" {
		t := syscall.StringToUTF16(title)
		titleP = u16p(t)
	}
	if initialDir != "" {
		d := syscall.StringToUTF16(initialDir)
		dirP = u16p(d)
	}
	if defExt != "" {
		e := syscall.StringToUTF16(defExt)
		extP = u16p(e)
	}

	flags := uint32(OFN_EXPLORER | OFN_PATHMUSTEXIST | OFN_NOCHANGEDIR)
	if save {
		flags |= OFN_OVERWRITEPROMPT
	} else {
		flags |= OFN_FILEMUSTEXIST | OFN_HIDEREADONLY
	}

	ofn := OPENFILENAMEW{
		HwndOwner:       owner,
		LpstrFilter:     u16p(ft),
		NFilterIndex:    1,
		LpstrFile:       &fileBuf[0],
		NMaxFile:        uint32(len(fileBuf)),
		LpstrTitle:      titleP,
		LpstrInitialDir: dirP,
		Flags:           flags,
		LpstrDefExt:     extP,
	}
	ofn.LStructSize = uint32(unsafe.Sizeof(ofn))

	var ok int32
	if save {
		ok = GetSaveFileNameW(&ofn)
	} else {
		ok = GetOpenFileNameW(&ofn)
	}
	if ok == 0 {
		return "", false
	}
	return syscall.UTF16ToString(fileBuf), true
}

// OpenFile 弹出「打开文件」对话框。filter 形如 "图片|*.png;*.jpg|所有文件|*.*"。
func OpenFile(owner uintptr, title, filter string) (string, bool) {
	return openFileDialog(owner, title, filter, "", "", false)
}

// OpenFiles 弹出可多选的「打开文件」对话框，返回选中的全部路径。
// 系统把多选结果存成「目录\0名1\0名2\0\0」，单选则是完整路径。
func OpenFiles(owner uintptr, title, filter string) ([]string, bool) {
	buf := make([]uint16, 32768)
	ft := FileDialogFilter(filter)
	var titleP *uint16
	if title != "" {
		t := syscall.StringToUTF16(title)
		titleP = u16p(t)
	}
	ofn := OPENFILENAMEW{
		HwndOwner:    owner,
		LpstrFilter:  u16p(ft),
		NFilterIndex: 1,
		LpstrFile:    &buf[0],
		NMaxFile:     uint32(len(buf)),
		LpstrTitle:   titleP,
		Flags:        OFN_EXPLORER | OFN_PATHMUSTEXIST | OFN_NOCHANGEDIR | OFN_FILEMUSTEXIST | OFN_HIDEREADONLY | OFN_ALLOWMULTISELECT,
	}
	ofn.LStructSize = uint32(unsafe.Sizeof(ofn))
	if GetOpenFileNameW(&ofn) == 0 {
		return nil, false
	}
	// 按 NUL 切分
	var parts []string
	start := 0
	for i, c := range buf {
		if c == 0 {
			if i == start {
				break // 双 NUL 结束
			}
			parts = append(parts, syscall.UTF16ToString(buf[start:i]))
			start = i + 1
		}
	}
	if len(parts) == 0 {
		return nil, false
	}
	if len(parts) == 1 {
		return parts, true
	}
	dir := parts[0]
	out := make([]string, 0, len(parts)-1)
	for _, name := range parts[1:] {
		if strings.ContainsAny(name, `\/`) {
			out = append(out, name) // 已是完整路径
			continue
		}
		out = append(out, strings.TrimRight(dir, `\/`)+`\`+name)
	}
	return out, len(out) > 0
}

// SaveFile 弹出「保存文件」对话框。
func SaveFile(owner uintptr, title, filter, defExt string) (string, bool) {
	return openFileDialog(owner, title, filter, "", defExt, true)
}

// PickFolder 弹出「选择文件夹」对话框。
func PickFolder(owner uintptr, title string) (string, bool) {
	display := make([]uint16, 260)
	var titleP *uint16
	if title != "" {
		t := syscall.StringToUTF16(title)
		titleP = u16p(t)
	}
	bi := BROWSEINFOW{
		HwndOwner:      owner,
		PszDisplayName: &display[0],
		LpszTitle:      titleP,
		UlFlags:        BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE,
	}
	pidl := SHBrowseForFolderW(&bi)
	if pidl == 0 {
		return "", false
	}
	defer CoTaskMemFree(pidl)
	path := make([]uint16, 1024)
	if SHGetPathFromIDListW(pidl, &path[0]) == 0 {
		return "", false
	}
	return syscall.UTF16ToString(path), true
}

// ShellOpen 用系统默认程序打开文件/网址。
func ShellOpen(path string) {
	p, _ := syscall.UTF16PtrFromString(path)
	op, _ := syscall.UTF16PtrFromString("open")
	ShellExecuteW(0, op, p, nil, nil, SW_SHOW)
}
