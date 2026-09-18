//go:build !windows

package screen

import "fmt"

// Region 框选区域（虚拟屏幕物理像素坐标）
type Region struct {
	X, Y, W, H int
}

// Empty 区域是否为空（未设置或已取消）
func (r Region) Empty() bool {
	return r.W <= 0 || r.H <= 0
}

// Normalized 返回规范化（左上/右下）后的矩形
func (r Region) Normalized() Region {
	x, y, w, h := r.X, r.Y, r.W, r.H
	if w < 0 {
		x += w
		w = -w
	}
	if h < 0 {
		y += h
		h = -h
	}
	return Region{X: x, Y: y, W: w, H: h}
}

// SelectRegion 框选遮罩（Windows 专属，其他平台暂不支持）
func SelectRegion() (Region, error) {
	return Region{}, fmt.Errorf("当前平台暂不支持框选区域")
}
