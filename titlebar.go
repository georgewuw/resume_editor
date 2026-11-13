package main

// 窗口顶栏配置，对应 MyGit 仓库根级的 titlebar.go。
// MyGit 是纯 Go 原生 UI，用 c.TitleBar() 绘制；本项目是 webview 前端，
// 因此这里只负责「隐藏原生标题栏」，顶栏由 frontend/styles/titlebar.css 绘制。

import "github.com/egoist/mygo"

// titleBarHeight 与 frontend/styles/base.css 的 --titlebar-h 保持一致。
const titleBarHeight = 40

// customTitleBar 隐藏系统标题栏与系统窗口按钮，交由页面自绘顶栏；
// 相比 Frameless，它保留 Windows 的阴影、圆角与四边可缩放边框。
// 页面用 --app-region: drag 标记拖拽区，用 mygo.window.* 驱动窗口。
func customTitleBar(o mygo.WindowOptions) mygo.WindowOptions {
	o.TitleBarStyle = mygo.TitleBarHidden
	o.TitleBarHeight = titleBarHeight
	o.HideWindowButtons = true
	// macOS 上与 MyGit 保持一致的红黄灯安全区（按钮已隐藏，仅作对齐基准）。
	o.TrafficLightPosition = &mygo.Point{X: 16, Y: (titleBarHeight - 14) / 2}
	return o
}
