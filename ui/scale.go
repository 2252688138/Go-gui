package ui

import (
	"os"
	"strconv"
)

// 界面缩放（整体 UI zoom）：
//
// 逻辑像素 = 物理像素 / (DPI 比例 × uiScale)，同时把 DPI 比例 × uiScale 传给
// 渲染器与字体缓存，因此布局、文字、控件、裁剪、命中测试全部同步放大/缩小，
// 不需要逐个组件改尺寸。鼠标事件来自 win32 层（已按 DPI 转逻辑像素），
// 这里再除以 uiScale 换算到缩放后的逻辑坐标系。
//
// 超高清（整帧超采样 SSAA）在缩放之上再多乘一层 ssaa：
// 渲染目标 = 物理像素 × ssaa，字形也按 ssaa 倍光栅化，最后降采样回窗口像素。
// 逻辑坐标、布局、命中测试完全不变，只有渲染分辨率变化。
//
// 降采样会轻微压低 1px 级细节的对比度（主观上“软”），因此 resolve 阶段
// 额外做一道轻度锐化（gpu.ssaaSharpen，可用 GOGUI_SS_SHARPEN 调节）。

var uiScale float32 = 1

const (
	UIScaleMin float32 = 0.6
	UIScaleMax float32 = 1.5
)

// 超高清（整帧超采样）档位：SSAuto = 自动，SSOff = 关闭，其余为显式倍数。
const (
	SSAuto float32 = 0
	SSOff  float32 = 1
	SS125  float32 = 1.25
	SS150  float32 = 1.5
	SS200  float32 = 2
)

// ssPixelBudget 超采样离屏目标的像素预算（约 8K：8192×4320）。
// 4K 窗口 × 2 倍已经是 7680×4320（RGBA 约 130MB），再往上会明显吃显存、压填充率，
// 因此超出预算时自动逐级降档（2× → 1.5× → 1.25× → 关），保证不会一开超高清就卡顿。
const ssPixelBudget = 8192 * 4320

// ssMode 超采样档位设置（SSAuto = 跟随 DPI 自动）。
var ssMode float32 = SSAuto

// init 允许用环境变量 GOGUI_SS 指定启动档位（0=自动 1=标准 1.25 1.5 2），
// 便于命令行 / 自动化对比不同清晰度下的画面，不影响正常启动（未设置时走自动）。
func init() {
	if v := os.Getenv("GOGUI_SS"); v != "" {
		if f, err := strconv.ParseFloat(v, 32); err == nil && f >= 0 && f <= 2 {
			ssMode = float32(f)
		}
	}
}

// UIScale 当前界面缩放（1 = 100%）。
func UIScale() float32 { return uiScale }

// SetUIScale 设置界面缩放（自动夹取到 60%~150%），立即重建渲染尺寸与字形。
func SetUIScale(k float32) {
	if k < UIScaleMin {
		k = UIScaleMin
	}
	if k > UIScaleMax {
		k = UIScaleMax
	}
	if k == uiScale {
		return
	}
	uiScale = k
	for _, a := range snapshotApps() {
		a.applyUIScale()
	}
	RequestRedraw()
}

// SetSuperSample 设置超高清档位（SSAuto / SSOff / SS125 / SS150 / SS200）。
// 立即重建渲染目标与字形图集，可运行时热切换（例如设置面板里切换清晰度）。
func SetSuperSample(k float32) {
	if k < 0 {
		k = 0
	} else if k > 0 && k < 1 {
		k = 1
	} else if k > 2 {
		k = 2
	}
	if k == ssMode {
		return
	}
	ssMode = k
	for _, a := range snapshotApps() {
		a.applyRenderScale()
	}
	RequestRedraw()
}

// SuperSampleMode 返回当前档位设置（SSAuto 表示跟随 DPI 自动）。
func SuperSampleMode() float32 { return ssMode }

// SuperSample 返回当前窗口实际生效的超采样倍数（含 DPI 自动判定，不含像素预算降档）。
func SuperSample() float32 {
	dpi := float32(1)
	if a := currentApp.Load(); a != nil && a.win != nil {
		dpi = a.win.Scale()
	}
	return superSampleFor(dpi, ssMode)
}

// superSampleFor 按 DPI 与档位算出超采样倍数。
// 自动档：DPI < 150% 时窗口像素本身比较粗（逻辑 1px 只对应 1~1.25 物理像素），
// 走 2× 超采样收益最大；150% 以上逻辑像素已对应 1.5+ 物理像素，1:1 绘制就足够清晰。
func superSampleFor(dpiScale, mode float32) float32 {
	if mode <= 0 {
		if dpiScale >= 1.5 {
			return 1
		}
		return 2
	}
	if mode < 1 {
		return 1
	}
	if mode > 2 {
		return 2
	}
	return mode
}

// effScale 渲染/字体实际使用的缩放（DPI × 界面缩放），不含超采样。
func (a *App) effScale() float32 {
	if a == nil || a.win == nil {
		return 1
	}
	s := a.win.Scale() * uiScale
	if s <= 0 {
		return 1
	}
	return s
}

// ssaaForBudget 按像素预算逐级降档：窗口太大时宁可少一点超采样也不让帧率崩掉。
// 纯函数，便于单测（renderDims 内部使用）。
func ssaaForBudget(pw, ph int32, ss float32) float32 {
	for ss > 1.01 && float64(pw)*float64(ph)*float64(ss)*float64(ss) > ssPixelBudget {
		switch {
		case ss > 1.5:
			ss = 1.5
		case ss > 1.25:
			ss = 1.25
		default:
			ss = 1
		}
	}
	return ss
}

// renderDims 本帧渲染目标的像素尺寸（含超采样）与实际生效的超采样倍数。
// 逻辑尺寸 = pw/scale = 物理像素/(DPI×uiScale)，与超采样倍数无关。
func (a *App) renderDims() (pw, ph int32, ss float32) {
	if a == nil || a.win == nil {
		return 1, 1, 1
	}
	dpw, dph := a.win.PixelSize()
	ss = ssaaForBudget(dpw, dph, superSampleFor(a.win.Scale(), ssMode))
	return int32(float32(dpw)*ss + 0.5), int32(float32(dph)*ss + 0.5), ss
}

// textureSS 光栅纹理（图表曲线等）应乘的超采样倍数。
// 与整帧 SSAA 同步，但不含像素预算降档后的影响——预算降档只影响离屏目标，
// 纹理分辨率仍按档位生成，避免「降档后曲线反而更糊」。
func (a *App) textureSS() float32 {
	if a == nil || a.r == nil {
		return 1
	}
	if ss := a.r.SuperSample(); ss > 1.01 {
		return ss
	}
	return 1
}

// renderSize 渲染目标尺寸与渲染缩放（DPI × 界面缩放 × 超采样）。
func (a *App) renderSize() (pw, ph int32, scale, ss float32) {
	pw, ph, ss = a.renderDims()
	return pw, ph, a.effScale() * ss, ss
}

// applyUIScale 缩放变化后重建渲染目标与字体缓存。
func (a *App) applyUIScale() {
	a.applyRenderScale()
}

// applyRenderScale 按当前 DPI / 界面缩放 / 超采样档位重建渲染目标、字形缓存与合成表面。
func (a *App) applyRenderScale() {
	if a == nil || a.win == nil || a.r == nil {
		return
	}
	a.win.MakeCurrent()
	pw, ph, scale, ss := a.renderSize()
	a.r.SetSuperSample(ss)
	a.r.Resize(pw, ph, scale)
	if a.fonts != nil {
		a.fonts.setScale(scale)
	}
	if a.win.HasCompose() {
		dpw, dph := a.win.PixelSize()
		a.win.ResizeCompose(dpw, dph)
	}
	a.dirty = true
}
