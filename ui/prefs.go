// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import (
	"encoding/json"
	"strconv"
	"sync"
	"time"
)

// 轻量偏好存储：Split 比例、列宽、Dock 布局、快捷键绑定等需要跨启动保留的状态。
//
//	ui.SetPref("editor.fontSize", "14")
//	ui.SavePrefs()                       // 立即落盘（一般不必手动调用）
//	size := ui.GetPrefFloat("editor.fontSize", 13)
//
// 写入去抖 400ms，拖动分隔条时不会频繁写盘；落盘为原子替换，损坏文件不会导致崩溃。

var (
	prefsMu       sync.Mutex
	prefsData     = map[string]string{}
	prefsLoaded   bool
	prefsDirty    bool
	prefsTimer    *time.Timer
	prefsDebounce = 400 * time.Millisecond
	prefsFile     string
)

// SetPref 写入一项偏好（值统一按字符串保存）。
func SetPref(key, val string) {
	if key == "" {
		return
	}
	prefsMu.Lock()
	ensurePrefsLocked()
	if prefsData[key] == val {
		prefsMu.Unlock()
		return
	}
	prefsData[key] = val
	prefsDirty = true
	prefsMu.Unlock()
	schedulePrefsSave()
}

// GetPref 读取一项偏好。
func GetPref(key string) (string, bool) {
	prefsMu.Lock()
	ensurePrefsLocked()
	v, ok := prefsData[key]
	prefsMu.Unlock()
	return v, ok
}

// DelPref 删除一项偏好。
func DelPref(key string) {
	prefsMu.Lock()
	ensurePrefsLocked()
	if _, ok := prefsData[key]; ok {
		delete(prefsData, key)
		prefsDirty = true
	}
	prefsMu.Unlock()
	schedulePrefsSave()
}

// GetPrefFloat 读取浮点偏好（缺失或解析失败时返回 def）。
func GetPrefFloat(key string, def float32) float32 {
	if v, ok := GetPref(key); ok {
		if f, err := strconv.ParseFloat(v, 32); err == nil {
			return float32(f)
		}
	}
	return def
}

// GetPrefInt 读取整数偏好。
func GetPrefInt(key string, def int) int {
	if v, ok := GetPref(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// GetPrefBool 读取布尔偏好（"1"/"true" 为真）。
func GetPrefBool(key string, def bool) bool {
	if v, ok := GetPref(key); ok {
		return v == "1" || v == "true" || v == "TRUE"
	}
	return def
}

// SetPrefFloat 写入浮点偏好。
func SetPrefFloat(key string, v float32) { SetPref(key, strconv.FormatFloat(float64(v), 'f', -1, 32)) }

// SetPrefInt 写入整数偏好。
func SetPrefInt(key string, v int) { SetPref(key, strconv.Itoa(v)) }

// SetPrefBool 写入布尔偏好。
func SetPrefBool(key string, v bool) {
	if v {
		SetPref(key, "1")
		return
	}
	SetPref(key, "0")
}

// PrefsPath 偏好文件路径（Web 端为空，表示用 localStorage）。
func PrefsPath() string {
	prefsMu.Lock()
	ensurePrefsLocked()
	p := prefsFile
	prefsMu.Unlock()
	return p
}

func ensurePrefsLocked() {
	if prefsLoaded {
		return
	}
	prefsLoaded = true
	scheduleLoadLocked()
}

// scheduleLoadLocked 延迟到第一次读写时再加载（避免包初始化就碰磁盘）。
func scheduleLoadLocked() {
	raw, path, err := prefsLoadRaw()
	prefsFile = path
	if err != nil || len(raw) == 0 {
		return
	}
	var m map[string]string
	if json.Unmarshal(raw, &m) == nil {
		for k, v := range m {
			prefsData[k] = v
		}
	}
}

func schedulePrefsSave() {
	prefsMu.Lock()
	if prefsTimer != nil {
		prefsTimer.Stop()
	}
	prefsTimer = time.AfterFunc(prefsDebounce, func() {
		_ = SavePrefs()
	})
	prefsMu.Unlock()
}

// SavePrefs 立即把偏好写入磁盘（原子替换）。
func SavePrefs() error {
	prefsMu.Lock()
	if !prefsDirty {
		prefsMu.Unlock()
		return nil
	}
	prefsDirty = false
	buf, err := json.MarshalIndent(prefsData, "", "  ")
	prefsMu.Unlock()
	if err != nil {
		return err
	}
	return prefsSaveRaw(buf)
}

// ResetPrefs 清空内存中的偏好（测试 / 恢复默认设置）。
func ResetPrefs() {
	prefsMu.Lock()
	prefsData = map[string]string{}
	prefsDirty = true
	prefsLoaded = true
	prefsMu.Unlock()
	schedulePrefsSave()
}
