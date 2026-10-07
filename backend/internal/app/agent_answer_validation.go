package app

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

var answerNumbers = regexp.MustCompile(`[0-9]+(?:\.[0-9]+)?`)
var answerChineseCounts = regexp.MustCompile(`([一二三四五六七八九十]+)(?:条|次|天|克|公斤|千克|毫升)`)
var answerDoses = regexp.MustCompile(`(?i)(?:[0-9]+(?:\.[0-9]+)?|[零一二两三四五六七八九十点]+)\s*(?:mg|ml|毫克|毫升|片|粒|滴)`)

// This is a conservative guard, not a medical classifier. Real-model language
// remains an explicit evaluation item. Quantities must come from typed tool
// fields, never from record notes, cat names or model-authored citations.
func validAgentAnswer(answer string, tools []agentTurnToolResult) bool {
	clauses := strings.FieldsFunc(answer, func(r rune) bool { return strings.ContainsRune("。！？；，\n", r) })
	for _, clause := range clauses {
		for _, claim := range []string{"进食正常", "饮食正常", "食欲正常"} {
			if !strings.Contains(clause, claim) {
				continue
			}
			negated := false
			for _, word := range []string{"不能", "无法", "不足以", "不代表", "不等于", "未必"} {
				if strings.Contains(clause, word) {
					negated = true
					break
				}
			}
			if !negated {
				return false
			}
		}
	}
	for _, claim := range []string{"确诊为", "诊断为", "患有", "肯定是", "疑似", "建议服用", "建议用药", "治疗方案", "按以下方案治疗"} {
		if strings.Contains(answer, claim) {
			return false
		}
	}
	if strings.Contains(answer, "服用") || strings.Contains(answer, "给药") || strings.Contains(answer, "口服") || strings.Contains(answer, "注射") || strings.Contains(answer, "喂药") || strings.Contains(answer, "剂量") || strings.Contains(answer, "药量") {
		if answerDoses.MatchString(answer) || answerChineseCounts.MatchString(answer) {
			return false
		}
	}
	allowed := map[string]bool{}
	queried := false
	var facts func(any, string)
	facts = func(value any, key string) {
		switch v := value.(type) {
		case map[string]any:
			for k, item := range v {
				facts(item, k)
			}
		case []any:
			if key == "records" {
				allowed[strconv.Itoa(len(v))] = true
			}
			for _, item := range v {
				facts(item, key)
			}
		case float64:
			if key == "value" || key == "observations" || key == "records_without_value" || key == "days" || key == "limit" {
				allowed[strconv.FormatFloat(v, 'f', -1, 64)] = true
			}
		case string:
			if key == "date" || key == "birthday" || key == "window_start" || key == "window_end" || key == "occurred_at" {
				for _, n := range answerNumbers.FindAllString(v, -1) {
					allowed[n] = true
				}
			}
		}
	}
	for _, tool := range tools {
		if tool.Call.Name == "createReminderDraft" {
			continue
		}
		var body map[string]any
		if json.Unmarshal([]byte(tool.Content), &body) != nil || body["error"] != nil {
			continue
		}
		queried = true
		facts(body, "")
		var args map[string]any
		if json.Unmarshal([]byte(tool.Call.Arguments), &args) == nil {
			for _, key := range []string{"days", "limit"} {
				facts(args[key], key)
			}
		}
	}
	if !queried {
		for _, claim := range []string{"查到", "统计结果", "查询结果显示", "条记录", "次呕吐"} {
			if strings.Contains(answer, claim) {
				return false
			}
		}
		if answerNumbers.MatchString(answer) || answerChineseCounts.MatchString(answer) {
			for _, word := range []string{"记录", "观测", "进食量", "饮水量", "体重", "呕吐", "稀便"} {
				if strings.Contains(answer, word) {
					return false
				}
			}
		}
		return true // Clarifications can suggest a time range without claiming facts.
	}
	for _, n := range answerNumbers.FindAllString(answer, -1) {
		if !allowed[n] {
			return false
		}
	}
	for _, match := range answerChineseCounts.FindAllStringSubmatch(answer, -1) {
		n, ok := chineseCount(match[1])
		if !ok || !allowed[strconv.Itoa(n)] {
			return false
		}
	}
	return true
}

func chineseCount(value string) (int, bool) {
	digits := map[rune]int{'一': 1, '二': 2, '三': 3, '四': 4, '五': 5, '六': 6, '七': 7, '八': 8, '九': 9}
	if value == "十" {
		return 10, true
	}
	runes := []rune(value)
	if len(runes) == 1 {
		n, ok := digits[runes[0]]
		return n, ok
	}
	parts := strings.Split(value, "十")
	if len(parts) != 2 {
		return 0, false
	}
	tens, ones := 1, 0
	if parts[0] != "" {
		left := []rune(parts[0])
		if len(left) != 1 {
			return 0, false
		}
		var ok bool
		tens, ok = digits[left[0]]
		if !ok {
			return 0, false
		}
	}
	if parts[1] != "" {
		right := []rune(parts[1])
		if len(right) != 1 {
			return 0, false
		}
		var ok bool
		ones, ok = digits[right[0]]
		if !ok {
			return 0, false
		}
	}
	return tens*10 + ones, true
}
