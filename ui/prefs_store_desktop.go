//go:build !js

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"os"
	"path/filepath"
	"strings"
)

// prefsLoadRaw 读取偏好文件（不存在时返回 nil, path, nil）。
func prefsLoadRaw() ([]byte, string, error) {
	path := defaultPrefsPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, path, nil
		}
		return nil, path, err
	}
	return raw, path, nil
}

// prefsSaveRaw 原子写入偏好文件（先写 .tmp 再重命名）。
func prefsSaveRaw(buf []byte) error {
	path := defaultPrefsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// defaultPrefsPath 偏好文件路径：用户配置目录/gogui/prefs.json。
func defaultPrefsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		dir = "."
	}
	name := "default"
	if len(os.Args) > 0 && os.Args[0] != "" {
		base := filepath.Base(os.Args[0])
		ext := filepath.Ext(base)
		name = strings.TrimSuffix(base, ext)
	}
	return filepath.Join(dir, "gogui", name+"-prefs.json")
}
