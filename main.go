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
		mygo.NewWindow(customTitleBar(mygo.WindowOptions{
			Title:     "简历编辑器",
			URL:       "/",
			Width:     1440,
			Height:    920,
			MinWidth:  1080,
			MinHeight: 680,
			StateKey:  "main",
		}))
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
