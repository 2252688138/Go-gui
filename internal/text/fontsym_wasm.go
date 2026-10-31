//go:build js && wasm

// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package text

// platformSymbolFamily 浏览器端没有「系统符号字体」概念，
// 字形覆盖由 canvas 自身的 font-family 回退链负责，这里返回空串表示不额外加载。
func platformSymbolFamily() string { return "" }
