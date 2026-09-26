//go:build !windows && !darwin

package audio

import (
	"context"
	"fmt"
)

// otherCapturer 未支持平台的占位实现
type otherCapturer struct {
	source Source
}

func newPlatformCapturer(src Source) (Capturer, error) {
	return nil, fmt.Errorf("当前平台不支持音频采集")
}

func (o *otherCapturer) Start(ctx context.Context) (<-chan []int16, <-chan error, error) {
	return nil, nil, fmt.Errorf("当前平台不支持音频采集")
}
