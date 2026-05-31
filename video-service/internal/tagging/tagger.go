package tagging

import (
	"strings"
	"unicode"
)

var keywordTags = map[string][]string{
	"go":        {"Go", "编程", "后端"},
	"gin":       {"Gin", "Web", "后端"},
	"grpc":      {"gRPC", "微服务", "后端"},
	"并发":        {"Go", "并发", "编程"},
	"goroutine": {"Go", "并发", "编程"},
	"channel":   {"Go", "并发", "编程"},
	"微服务":       {"微服务", "后端", "架构"},
	"api":       {"Web", "后端"},
	"rest":      {"Web", "REST", "后端"},
	"docker":    {"Docker", "DevOps", "部署"},
	"kafka":     {"Kafka", "消息队列", "后端"},
	"redis":     {"Redis", "缓存", "后端"},
	"mysql":     {"MySQL", "数据库", "后端"},
	"mongo":     {"MongoDB", "数据库", "后端"},
	"vue":       {"Vue", "前端", "Web"},
	"python":    {"Python", "编程"},
	"教程":        {"教程", "入门"},
	"入门":        {"教程", "入门"},
	"实战":        {"实战", "项目"},
	"框架":        {"框架", "Web"},
}

func Extract(title, description string) []string {
	text := strings.ToLower(title + " " + description)
	seen := make(map[string]bool)
	out := make([]string, 0, 8)
	add := func(tag string) {
		tag = strings.TrimSpace(tag)
		if tag == "" || seen[tag] {
			return
		}
		seen[tag] = true
		out = append(out, tag)
	}
	for kw, tags := range keywordTags {
		if strings.Contains(text, kw) {
			for _, t := range tags {
				add(t)
			}
		}
	}
	for _, token := range splitTokens(title + " " + description) {
		if len([]rune(token)) >= 2 && len([]rune(token)) <= 12 {
			add(token)
		}
	}
	if len(out) == 0 {
		add("短视频")
	}
	if len(out) > 8 {
		out = out[:8]
	}
	return out
}

func MergeTags(primary, fallback []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, 8)
	add := func(tag string) {
		tag = strings.TrimSpace(tag)
		if tag == "" || seen[tag] {
			return
		}
		seen[tag] = true
		out = append(out, tag)
	}
	for _, t := range primary {
		add(t)
	}
	for _, t := range fallback {
		add(t)
	}
	if len(out) > 8 {
		out = out[:8]
	}
	return out
}

func splitTokens(s string) []string {
	s = strings.ToLower(s)
	var b strings.Builder
	tokens := make([]string, 0)
	flush := func() {
		if b.Len() > 0 {
			tokens = append(tokens, b.String())
			b.Reset()
		}
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r > 127 {
			b.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return tokens
}
