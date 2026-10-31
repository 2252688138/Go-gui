// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "gogui/internal/geom"

// fitVideoRect 按填充方式计算画面实际矩形（Contain 留黑边 / Cover 溢出裁切 / Fill 拉伸）。
//
// 纯几何计算，与平台无关，因此放在无构建标签的文件里：
// 视频组件在所有端共用同一套缩放逻辑，video_test.go 也能在各平台跑。
func fitVideoRect(r geom.Rect, sw, sh float32, fit VideoFit) geom.Rect {
	if sw <= 0 || sh <= 0 || r.W <= 0 || r.H <= 0 {
		return r
	}
	if fit == FitFill {
		return r
	}
	scale := r.W / sw
	if fit == FitCover {
		if v := r.H / sh; v > scale {
			scale = v
		}
	} else if v := r.H / sh; v < scale {
		scale = v
	}
	dw, dh := sw*scale, sh*scale
	return geom.Rect{X: r.X + (r.W-dw)/2, Y: r.Y + (r.H-dh)/2, W: dw, H: dh}
}
