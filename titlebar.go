package main

// 窗口顶栏由网页绘制，此处配置原生窗口与网页的衔接。

import "github.com/egoist/mygo"

// 原生按钮占用的高度与左右空间由 mygo 的 CSS 变量提供给网页。
const titleBarHeight = 40

// customTitleBar keeps each desktop's native controls over the custom toolbar:
// traffic lights on macOS and the desktop's title buttons on Windows and Linux.
func customTitleBar(o mygo.WindowOptions) mygo.WindowOptions {
	o.TitleBarStyle = mygo.TitleBarHidden
	o.TitleBarHeight = titleBarHeight
	o.HideWindowButtons = false
	o.TrafficLightPosition = &mygo.Point{X: 16, Y: (titleBarHeight - 14) / 2}
	return o
}
