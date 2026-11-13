package main

import (
	"log"

	"github.com/egoist/mygo"
)

func main() {
	svc, err := newResumeService()
	if err != nil {
		log.Fatal(err)
	}
	// 服务类型名仍为 Resume，前端 window.mygo.call("Resume.*") 无需改动。
	mygo.Bind(svc)

	mygo.App.WhenReady(func() {
		// 启动尺寸按主显示器工作区推算，见 windowsize.go。
		width, height := initialWindowSize()
		mygo.NewWindow(customTitleBar(mygo.WindowOptions{
			Title:     "简历编辑器",
			URL:       "/",
			Width:     width,
			Height:    height,
			MinWidth:  minInitialWidth,
			MinHeight: minInitialHeight,
			StateKey:  "main",
		}))
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
