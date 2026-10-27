//go:build windows

package win32

// 合成控制器（B 阶段）相关槽位：
//
//	ICoreWebView2Environment3::CreateCoreWebView2CompositionController = 3
//	ICoreWebView2CompositionController::put_RootVisualTarget = 4
//	ICoreWebView2CompositionController::SendMouseInput = 5
//	ICoreWebView2CompositionController2::get_UiController = 11
//
// 注意：WebView2 的虚表顺序与 dcomp 不同，这里按 WebView2.h 的声明顺序。
const (
	wvEnvCompCreate = 3
	wvCompPutRoot   = 4
	wvCompSendMouse = 5
	wvCompGetUI     = 11
)
