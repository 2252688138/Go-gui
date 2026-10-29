// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

// Package version 是应用版本号的唯一来源。
//
// 版本号写在仓库根目录的 VERSION 文件里（一行，语义化版本 x.y.z），
// 由 build.ps1 读取后通过链接期变量注入：
//
//	go build -ldflags "-X gogui/internal/version.Number=1.2.3 -X gogui/internal/version.Commit=abc1234 ..."
//
// 未走构建脚本时（go run / go test / IDE）使用本文件里的默认值，
// 因此业务代码只依赖 version.Number，不需要自己解析 VERSION 文件。
//
// 注意：包目录不能命名为 version（与根目录 VERSION 文件在 Windows 上重名）。
package version

import (
	"fmt"
	"strconv"
	"strings"
)

// 以下变量由 -ldflags -X 在构建期覆盖，业务代码只读不写。
var (
	// Number 语义化版本号 x.y.z，可带预发布后缀（如 1.2.0-rc1）。
	Number = "0.1.0"
	// Product 产品名：exe 文件属性、安装器、关于窗口共用同一份。
	Product = "GoGUI"
	// Company 公司 / 作者，写入 exe 文件属性的「公司」项。
	Company = "Go-gui"
	// Copyright 版权声明，写入 exe 文件属性与安装器。
	Copyright = "Copyright (c) 2026 Mr.Shu"
	// Description 文件说明（exe 属性里的「文件说明」）。
	Description = "GoGUI 纯 Go 自绘 GUI 框架"
	// Commit 构建时的 git 提交短哈希，未取到为 none。
	Commit = "none"
	// BuildTime 构建时间（UTC，RFC3339），未注入为 unknown。
	BuildTime = "unknown"
	// Channel 发布通道：dev / beta / release。
	Channel = "dev"
)

// Info 版本信息快照，便于日志输出或 JSON 化。
type Info struct {
	Product   string `json:"product"`
	Number    string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"buildTime"`
	Channel   string `json:"channel"`
}

// Current 返回当前构建的版本信息快照。
func Current() Info {
	return Info{
		Product:   Product,
		Number:    Number,
		Commit:    Commit,
		BuildTime: BuildTime,
		Channel:   Channel,
	}
}

// String 返回带 v 前缀的版本号，如 "v1.2.3"。
func String() string { return "v" + Number }

// Title 返回窗口标题常用的 "产品名 v1.2.3"。
func Title() string { return Product + " " + String() }

// Full 返回完整描述，用于「关于」与日志：
// "v1.2.3 · release · 1a2b3c4 · 2026-10-29T02:11:00Z"。
func Full() string {
	parts := []string{String()}
	if Channel != "" {
		parts = append(parts, Channel)
	}
	if Commit != "" && Commit != "none" {
		parts = append(parts, Commit)
	}
	if BuildTime != "" && BuildTime != "unknown" {
		parts = append(parts, BuildTime)
	}
	return strings.Join(parts, " · ")
}

// IsRelease 是否为正式发布通道（release 通道的构建才适合对外分发）。
func IsRelease() bool { return Channel == "release" }

// Parse 解析语义化版本 "1.2.3-rc1" → (1, 2, 3, "rc1")。
// 允许前缀 v；段数不足按 0 处理；非法字符返回错误。
func Parse(v string) (major, minor, patch int, pre string, err error) {
	s := strings.TrimSpace(v)
	s = strings.TrimPrefix(strings.TrimPrefix(s, "v"), "V")
	if s == "" {
		return 0, 0, 0, "", fmt.Errorf("version: 空版本号")
	}
	// 构建元数据（+build）不参与比较，直接丢弃
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	if i := strings.IndexByte(s, '-'); i >= 0 {
		pre = s[i+1:]
		s = s[:i]
	}
	nums := strings.Split(s, ".")
	if len(nums) > 3 {
		return 0, 0, 0, "", fmt.Errorf("version: 版本号段数过多 %q", v)
	}
	dst := []*int{&major, &minor, &patch}
	for i, n := range nums {
		if n == "" {
			return 0, 0, 0, "", fmt.Errorf("version: 版本号段为空 %q", v)
		}
		x, e := strconv.Atoi(n)
		if e != nil || x < 0 {
			return 0, 0, 0, "", fmt.Errorf("version: 非法版本号 %q", v)
		}
		*dst[i] = x
	}
	return major, minor, patch, pre, nil
}

// Compare 比较两个语义化版本：a<b 返回 -1，相等 0，a>b 返回 1。
// 遵循 semver：主次修订号逐段比较，带预发布后缀的版本低于同号正式版。
// 任一侧解析失败时退回字符串比较，保证不 panic。
func Compare(a, b string) int {
	am, ai, ap, apre, aerr := Parse(a)
	bm, bi, bp, bpre, berr := Parse(b)
	if aerr != nil || berr != nil {
		return strings.Compare(a, b)
	}
	if c := cmpInt(am, bm); c != 0 {
		return c
	}
	if c := cmpInt(ai, bi); c != 0 {
		return c
	}
	if c := cmpInt(ap, bp); c != 0 {
		return c
	}
	switch {
	case apre == bpre:
		return 0
	case apre == "": // 正式版 > 预发布版
		return 1
	case bpre == "":
		return -1
	default:
		return strings.Compare(apre, bpre)
	}
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// String 让 Info 直接满足 fmt.Stringer，日志里 %v 即可输出完整版本。
func (i Info) String() string {
	return fmt.Sprintf("%s v%s (%s, %s, %s)", i.Product, i.Number, i.Channel, i.Commit, i.BuildTime)
}
