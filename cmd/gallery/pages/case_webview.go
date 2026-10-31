// Go-gui — gallery 页面拆分：WebView 浏览器
// 本文件属于 pages 包，与其它页面文件一起由 main.go 引入。

package pages

import (
	"gogui/ui"
)

// ---- 案例：WebView 内嵌浏览器（系统 WebView2）----

var (
	wvInst  *ui.WebView
	wvAddr  = "https://www.bing.com"
	wvTried bool
	wvTip   string
)

// webviewDemoHTML 本机示例页（离线也能看、能交互：输入 / 按钮 / 列表滚动）。
const webviewDemoHTML = `<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8">
<title>GoGUI · WebView 示例</title>
<style>
  :root{color-scheme:dark}
  *{box-sizing:border-box}
  body{margin:0;font:14px/1.6 'Microsoft YaHei UI',system-ui,sans-serif;background:#0b1120;color:#e6ecff}
  .hero{padding:28px 30px;background:linear-gradient(120deg,#6366f1,#8b5cf6 55%,#22d3ee)}
  .hero h1{margin:0;font-size:24px;letter-spacing:.5px}
  .hero p{margin:6px 0 0;opacity:.9;font-size:13px}
  .wrap{padding:22px 30px 40px;display:grid;gap:16px;grid-template-columns:repeat(auto-fit,minmax(260px,1fr))}
  .card{background:#111a2c;border:1px solid #1f2a41;border-radius:14px;padding:16px}
  .card h2{margin:0 0 10px;font-size:14px;color:#a5b4fc;letter-spacing:.4px}
  input[type=text]{width:100%;padding:10px 12px;border-radius:9px;border:1px solid #2a3550;background:#0c1424;color:#e6ecff;outline:none}
  input[type=text]:focus{border-color:#6366f1;box-shadow:0 0 0 3px rgba(99,102,241,.25)}
  button{margin-top:10px;padding:9px 14px;border:0;border-radius:9px;background:linear-gradient(90deg,#6366f1,#8b5cf6);color:#fff;font-weight:600;cursor:pointer}
  button:active{transform:translateY(1px)}
  .row{display:flex;align-items:center;gap:10px;margin-top:8px;color:#93a2c4}
  .pill{display:inline-block;padding:3px 9px;border-radius:999px;background:rgba(99,102,241,.18);color:#c7d2fe;font-size:12px}
  .list{max-height:150px;overflow:auto;margin:0;padding-left:18px;color:#93a2c4}
  a{color:#7dd3fc}
  .bar{height:8px;border-radius:999px;background:#1b2436;overflow:hidden;margin-top:12px}
  .bar i{display:block;height:100%;width:38%;background:linear-gradient(90deg,#22d3ee,#8b5cf6);animation:slide 3s ease-in-out infinite}
  @keyframes slide{0%,100%{margin-left:0}50%{margin-left:62%}}
</style></head><body>
  <div class="hero">
    <h1>GoGUI · WebView2</h1>
    <p>这段页面由系统 Edge 内核渲染，滚动、输入、按钮、动画都是浏览器原生的。</p>
  </div>
  <div class="wrap">
    <div class="card">
      <h2>真实交互验证</h2>
      <input id="name" type="text" placeholder="输入点东西…">
      <button onclick="greet()">点我</button>
      <div class="row"><span class="pill" id="out">等待操作</span></div>
    </div>
    <div class="card">
      <h2>滚动与链接</h2>
      <ul class="list">
        <li>第一行：鼠标滚轮可滚动</li><li>第二行：中文输入法直接可用</li>
        <li>第三行：右键菜单、复制粘贴原生</li><li>第四行：视频 / canvas 均走 GPU</li>
        <li>第五行：链接可点击 <a href="https://www.bing.com">Bing</a></li>
        <li>第六行：重新加载、前进后退由自绘工具栏控制</li>
        <li>第七行：滚动到这里说明列表可滚</li><li>第八行：结束</li>
      </ul>
      <div class="bar"><i></i></div>
    </div>
  </div>
  <script>
    function greet(){
      var v = document.getElementById('name').value;
      document.getElementById('out').textContent = v ? ('你好，' + v + '！') : '先输入内容再点我';
    }
    document.getElementById('name').addEventListener('keydown', function(e){ if(e.key === 'Enter') greet(); });
  </script>
</body></html>`

// webviewCase WebView 组件演示页：自绘工具栏 + 真实网页。
func webviewCase() *ui.Node {
	if !wvTried {
		wvTried = true
		if !ui.WebViewAvailable() {
			wvTip = "未检测到 WebView2 运行时，先安装 Microsoft Edge WebView2 Runtime"
		} else if wv, err := ui.NewWebView(ui.WebViewOptions{HTML: webviewDemoHTML}); err != nil {
			wvTip = "创建 WebView 失败：" + err.Error()
		} else {
			wvInst = wv
			wv.SetRadius(12)
		}
	}

	header := ui.Row(
		ui.Column(
			ui.Text("WebView 内嵌浏览器").St(ui.St().Text(19, 700, "#e8ecff")),
			ui.Text("系统 WebView2（Edge 内核）· 网页由 Chromium 的 GPU 进程渲染，交互与帧率都是原生的。").
				St(ui.St().Text(11.5, 400, ui.ColTextMuted)),
		).Grow(1).St(ui.St().GapOf(4)),
		ui.Tag("A 阶段 · HWND 托管", "#7dd3fc"),
	).St(ui.St().GapOf(12)).AlignTo(ui.AlignCenter)

	var bar *ui.Node
	if wvInst != nil {
		bar = ui.WebViewBar(wvInst, ui.WebViewBarOptions{
			Address:   wvAddr,
			Key:       "gv-addr",
			OnAddress: func(v string) { wvAddr = v },
			OnGo:      func() { navigateWebView(wvAddr) },
			ShowTitle: true,
		})
	} else {
		bar = ui.Row(
			ui.Text("浏览器不可用").St(ui.St().Text(12, 600, "#f87171")),
		).St(ui.St().Padding(10).R(10).BgC(ui.ColBgSoft).Stroke(1, ui.ColStroke))
	}

	links := ui.Row(
		ui.Btn("本机示例", ui.BtnSecondary, func() {
			if wvInst != nil {
				wvInst.LoadHTML(webviewDemoHTML)
				wvTip = "已加载本机示例页（离线可交互）"
			}
		}),
		ui.Btn("Bing", ui.BtnSecondary, func() { navigateWebView("https://www.bing.com") }),
		ui.Btn("MDN", ui.BtnSecondary, func() { navigateWebView("https://developer.mozilla.org/zh-CN/") }),
		ui.Btn("GitHub", ui.BtnSecondary, func() { navigateWebView("https://github.com") }),
		ui.Spacer(),
		ui.Text(wvTip).St(ui.St().Text(11, 400, ui.ColTextMuted).WrapOn()),
	).St(ui.St().GapOf(8)).AlignTo(ui.AlignCenter)

	status := ui.Box().St(ui.St().H(1))
	var area *ui.Node
	if wvInst != nil {
		status = ui.WebViewStatus(wvInst)
		if TitleMenu.IsOpen() {
			// 菜单是自绘浮层，会被网页子窗口盖住，展开时先把网页临时隐藏
			area = ui.Center(
				ui.Column(
					ui.Text("菜单已展开 · 网页临时隐藏").St(ui.St().Text(13, 600, "#c7d2fe")),
					ui.Text("HWND 托管的网页永远在最上层，A 阶段无法被自绘浮层遮挡；升级到 DirectComposition 合成后可解除此限制。").
						St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn().TAlign(ui.AlignCenter)),
				).St(ui.St().GapOf(6)).AlignTo(ui.AlignCenter),
			).St(ui.St().GrowTo(1).R(12).BgC(ui.ColBgDeep).Stroke(1, ui.ColStroke))
		} else {
			area = ui.WebViewNode(wvInst).St(ui.St().GrowTo(1).R(12))
		}
	} else {
		area = ui.Center(
			ui.Column(
				ui.Text("WebView2 暂不可用").St(ui.St().Text(14, 600, "#f87171")),
				ui.Text(wvTip).St(ui.St().Text(11.5, 400, ui.ColTextMuted).WrapOn().TAlign(ui.AlignCenter)),
			).St(ui.St().GapOf(6)).AlignTo(ui.AlignCenter),
		).St(ui.St().GrowTo(1).R(12).BgC(ui.ColBgDeep).Stroke(1, ui.ColStroke))
	}

	return ui.Column(header, bar, links, status, area).
		St(ui.St().Padding(18).GapOf(12)).Grow(1)
}

// navigateWebView 地址栏 / 快捷入口统一走这里（补全协议 + 提示）。
func navigateWebView(url string) {
	if wvInst == nil {
		return
	}
	wvInst.Navigate(url)
	wvTip = "正在打开 " + url
	ui.RequestRedraw()
}
