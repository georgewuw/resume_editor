package main

// 启动窗口尺寸策略

import "github.com/egoist/mygo"

const (
	// 占主显示器工作区的百分比，为桌面边缘留出余地。
	initialWidthPercent  = 86
	initialHeightPercent = 90

	// 大屏上限：工作区再大也不超过此尺寸，避免窗口铺满。
	maxInitialWidth  = 1320
	maxInitialHeight = 880

	// 布局下限：低于此值三栏会互相挤压，与 base.css 的分栏几何一致。
	minInitialWidth  = 1080
	minInitialHeight = 680
)

// initialWindowSize 按主显示器工作区推算默认窗口尺寸，单位为 DIP，与
// WindowOptions.Width、Height 一致。取不到显示器信息时回退到上限值。
func initialWindowSize() (width, height int) {
	workArea := mygo.Screen.PrimaryDisplay().WorkArea
	if workArea.Width <= 0 || workArea.Height <= 0 {
		return maxInitialWidth, maxInitialHeight
	}
	return sizeForWorkArea(workArea.Width, workArea.Height)
}

// sizeForWorkArea 是尺寸策略中与显示器无关的部分，便于测试。
func sizeForWorkArea(areaWidth, areaHeight int) (width, height int) {
	width = clampInt(areaWidth*initialWidthPercent/100, minInitialWidth, maxInitialWidth)
	height = clampInt(areaHeight*initialHeightPercent/100, minInitialHeight, maxInitialHeight)
	// 工作区比下限还小时，优先保证窗口完整可见。
	if width > areaWidth {
		width = areaWidth
	}
	if height > areaHeight {
		height = areaHeight
	}
	return width, height
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
