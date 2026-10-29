//go:build js && wasm

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

// 浏览器版没有本地文件系统缓存，远程图片统一降级为占位卡。
// （图片管线的 ImageSize/DecodeImage 依赖 os.Open 读取本地路径。）

func mdRemoteImagePath(url string) (string, bool) { return "", false }

func mdRemoteError(url string) string {
	return "浏览器版不支持远程图片（需先下载到本地）"
}

func mdRemotePending(url string) bool { return false }
