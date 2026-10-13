package domain

import (
	"strings"
	"testing"
)

func TestScenePrompt(t *testing.T) {
	p := GetScenePrompt()
	t.Log("\n" + p)

	if !strings.HasPrefix(p, "<ScenePrompt>") || !strings.HasSuffix(p, "</ScenePrompt>") {
		t.Fatal("场景提示词应被 <ScenePrompt> 包裹")
	}

	seen := map[string]bool{}
	ids := make([]string, 0, len(scenes))
	for _, s := range scenes {
		if s.ID == "" || s.Name == "" || s.When == "" || strings.TrimSpace(s.Requirements) == "" {
			t.Fatalf("场景 %q 存在空字段", s.ID)
		}
		if seen[s.ID] {
			t.Fatalf("场景 ID 重复: %s", s.ID)
		}
		seen[s.ID] = true
		ids = append(ids, s.ID)
		if !strings.Contains(p, `<Scene id="`+s.ID+`"`) {
			t.Errorf("提示词缺少场景 %s", s.ID)
		}
	}

	// 优先级顺序按切片顺序输出，且兜底场景排在最后
	if !strings.Contains(p, strings.Join(ids, " > ")) {
		t.Error("提示词缺少按场景顺序生成的优先级说明")
	}
	if ids[len(ids)-1] != "general" {
		t.Error("兜底场景 general 应排在最后")
	}
}
