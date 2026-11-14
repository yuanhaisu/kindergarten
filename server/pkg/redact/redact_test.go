package redact

import "testing"

func TestPhone(t *testing.T) {
	cases := map[string]string{
		"13800138000": "138****8000",
		"138****8000": "138****8000",
		"13":          "13",
		"":            "",
	}
	for in, want := range cases {
		if got := Phone(in); got != want {
			t.Errorf("Phone(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIDCard(t *testing.T) {
	cases := map[string]string{
		"110101199001011234": "110101********1234",
		"110101********1234": "110101********1234",
		"11010119900101123X": "110101********123X",
		"123":                "123",
	}
	for in, want := range cases {
		if got := IDCard(in); got != want {
			t.Errorf("IDCard(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStudentNo(t *testing.T) {
	cases := map[string]string{
		"20240001": "20****01",
		"20****01": "20****01",
		"1234":     "1234",
	}
	for in, want := range cases {
		if got := StudentNo(in); got != want {
			t.Errorf("StudentNo(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestName(t *testing.T) {
	cases := map[string]string{
		"张三":   "张*",
		"欧阳娜娜": "欧***",
		"张*":   "张*",
		"A":    "A",
		"":     "",
	}
	for in, want := range cases {
		if got := Name(in); got != want {
			t.Errorf("Name(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestText(t *testing.T) {
	in := "家长 13800138000，身份证 110101199001011234，工单 20240001"
	want := "家长 138****8000，身份证 110101********1234，工单 20240001"
	if got := Text(in); got != want {
		t.Fatalf("Text() = %q, want %q", got, want)
	}

	// 幂等：对结果再执行一次不变化。
	if got := Text(want); got != want {
		t.Fatalf("Text() not idempotent: %q", got)
	}

	// 非个人信息数字不受影响。
	if got := Text("订单 12345678901 已完成"); got != "订单 12345678901 已完成" {
		t.Fatalf("Text() altered non-PII: %q", got)
	}
}
