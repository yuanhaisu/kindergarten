// Package redact 对写入操作日志的个人信息做掩码。
//
// 自由文本只处理手机号与身份证号（可识别、误伤低）；姓名与学号必须由
// 结构化写入点在拼接前调用 Name/StudentNo，不能靠正则猜。
package redact

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	phonePattern  = regexp.MustCompile(`1[3-9]\d{9}`)
	idCardPattern = regexp.MustCompile(`\d{17}[\dXx]`)
)

// Phone 保留前 3 后 4；长度不足 7 位原样返回。
func Phone(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 7 {
		return s
	}
	return s[:3] + strings.Repeat("*", len(s)-7) + s[len(s)-4:]
}

// IDCard 保留前 6 后 4；长度不足 10 位原样返回。
func IDCard(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 10 {
		return s
	}
	return s[:6] + strings.Repeat("*", len(s)-10) + s[len(s)-4:]
}

// StudentNo 保留前 2 后 2；长度不足 5 位原样返回。
func StudentNo(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 5 {
		return s
	}
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}

// Name 保留首字，其余替换为 *。
func Name(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	_, size := utf8.DecodeRuneInString(s)
	if size >= len(s) {
		return s
	}
	return s[:size] + strings.Repeat("*", utf8.RuneCountInString(s[size:]))
}

// Text 对自由文本中的手机号与身份证号做掩码；幂等。
func Text(s string) string {
	if s == "" {
		return s
	}
	// 先处理 18 位身份证，避免号码内部的连续数字被手机号规则误匹配。
	s = idCardPattern.ReplaceAllStringFunc(s, IDCard)
	s = phonePattern.ReplaceAllStringFunc(s, Phone)
	return s
}
