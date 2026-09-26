//go:build windows

package audio

import "testing"

func TestSystemVolume(t *testing.T) {
	vol, muted, err := SystemVolume()
	if err != nil {
		t.Fatalf("SystemVolume 失败: %v", err)
	}
	t.Logf("系统音量=%.2f muted=%v", vol, muted)
}
