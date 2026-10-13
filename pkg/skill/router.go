package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"Q-Solver/pkg/llm"
	"Q-Solver/pkg/logger"
)

const (
	routeTimeout  = 8 * time.Second // 路由超时：超时即视为未命中，不拖慢答题
	maxPicked     = 2               // 单题最多加载的技能数
	maxBodyRunes  = 12000           // 单个技能注入正文上限
	maxTotalRunes = 20000           // 注入总量上限
)

const routerPrompt = `你是技能路由器。下面是可用技能列表（id：说明）。根据用户的题目，选出确实有助于解答它的技能。

规则：
- 只在题目与技能说明明显相关时才选；拿不准就不选。
- 最多选 %d 个。
- 只输出一个 JSON 字符串数组，元素为技能 id，例如 ["algo-java"]；没有合适的技能输出 []。
- 不要输出任何其他文字。

可用技能：
%s`

var jsonArrayRe = regexp.MustCompile(`\[[^\[\]]*\]`)

// Select 用路由模型从候选技能中挑选与题目相关的技能。
// text 为文字题目（面试转写），images 为题目截图（搜题）；二者至少有一个。
// 任何失败（超时、解析失败、无命中）都返回空——技能是增强项，不能阻塞答题。
func Select(ctx context.Context, provider llm.Provider, model, text string, images []string, candidates []Skill) []Skill {
	if provider == nil || len(candidates) == 0 {
		return nil
	}
	if strings.TrimSpace(text) == "" && len(images) == 0 {
		return nil
	}

	var catalog strings.Builder
	for _, sk := range candidates {
		fmt.Fprintf(&catalog, "- %s：%s\n", sk.ID, sk.Description)
	}
	system := fmt.Sprintf(routerPrompt, maxPicked, catalog.String())

	var user llm.Message
	if len(images) > 0 {
		parts := []llm.ContentPart{llm.TextPart("题目见截图。")}
		for _, img := range images {
			parts = append(parts, llm.ImagePart(img))
		}
		user = llm.NewMultiPartMessage(llm.RoleUser, parts)
	} else {
		user = llm.NewUserMessage("题目：" + text)
	}

	rctx, cancel := context.WithTimeout(ctx, routeTimeout)
	defer cancel()
	start := time.Now()
	resp, err := provider.GenerateContent(rctx, model, []llm.Message{llm.NewSystemMessage(system), user})
	if err != nil {
		if ctx.Err() == nil {
			logger.Printf("[skill] 路由失败（%v），按未命中处理: %v", time.Since(start), err)
		}
		return nil
	}

	picked := pick(resp.Content, candidates)
	logger.Printf("[skill] 路由完成（%v）：%v", time.Since(start), Names(picked))
	return picked
}

// pick 解析路由输出里的 id 数组，按候选集过滤并限量
func pick(output string, candidates []Skill) []Skill {
	m := jsonArrayRe.FindString(output)
	if m == "" {
		return nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(m), &ids); err != nil {
		return nil
	}
	byID := make(map[string]Skill, len(candidates))
	for _, sk := range candidates {
		byID[sk.ID] = sk
	}
	var out []Skill
	seen := map[string]bool{}
	for _, id := range ids {
		sk, ok := byID[strings.TrimSpace(id)]
		if !ok || seen[sk.ID] {
			continue
		}
		seen[sk.ID] = true
		out = append(out, sk)
		if len(out) >= maxPicked {
			break
		}
	}
	return out
}

// Block 把选中的技能渲染为可追加到 system 提示词的片段（无技能返回空串）
func Block(picked []Skill) string {
	if len(picked) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n<Skills>\n")
	b.WriteString("  <Note>以下技能与当前题目相关，请遵循其中的方法、要点与输出要求作答；技能与上文输出规范冲突时，以上文输出规范为准。</Note>\n")
	total := 0
	for _, sk := range picked {
		body := truncateRunes(sk.Body, maxBodyRunes)
		if total+len([]rune(body)) > maxTotalRunes {
			break
		}
		total += len([]rune(body))
		fmt.Fprintf(&b, "  <Skill name=%q>\n%s\n  </Skill>\n", sk.Name, body)
	}
	b.WriteString("</Skills>\n")
	return b.String()
}

// Names 返回技能名称列表（用于前端提示）
func Names(picked []Skill) []string {
	out := make([]string, len(picked))
	for i, sk := range picked {
		out[i] = sk.Name
	}
	return out
}
