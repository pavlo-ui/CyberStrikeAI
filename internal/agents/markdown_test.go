package agents

import (
	"testing"
)

func TestSplitFrontMatter(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantFront string
		wantBody  string
		wantErr   bool
	}{
		{
			name:      "valid front matter and body",
			content:   "---\ntitle: test\n---\nbody content",
			wantFront: "title: test",
			wantBody:  "body content",
			wantErr:   false,
		},
		{
			name:      "missing front matter",
			content:   "just body content",
			wantFront: "",
			wantBody:  "just body content",
			wantErr:   false,
		},
		{
			name:      "missing closing delimiter",
			content:   "---\ntitle: test\nbody content",
			wantFront: "",
			wantBody:  "",
			wantErr:   true,
		},
		{
			name:      "empty body",
			content:   "---\ntitle: test\n---\n",
			wantFront: "title: test",
			wantBody:  "",
			wantErr:   false,
		},
		{
			name:      "front matter with CRLF",
			content:   "---\r\ntitle: test\r\n---\r\nbody content",
			wantFront: "title: test",
			wantBody:  "body content",
			wantErr:   false,
		},
		{
			name:      "extra spaces around",
			content:   "  \n---\ntitle: test\n---\nbody content \n ",
			wantFront: "title: test",
			wantBody:  "body content",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFront, gotBody, err := SplitFrontMatter(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitFrontMatter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotFront != tt.wantFront {
				t.Errorf("SplitFrontMatter() gotFront = [%v], want [%v]", gotFront, tt.wantFront)
			}
			if gotBody != tt.wantBody {
				t.Errorf("SplitFrontMatter() gotBody = [%v], want [%v]", gotBody, tt.wantBody)
			}
		})
	}
}

func TestSlugID(t *testing.T) {
	tests := []struct {
		name string
		input string
		want  string
	}{
		{"normal", "My Agent", "my-agent"},
		{"delimiters", "My_Agent/Test.Name", "my-agent-test-name"},
		{"multiple delimiters", "My   Agent --- Test", "my-agent-test"},
		{"non-ascii", "My Agent 🤖", "my-agent"},
		{"digits", "Agent 007", "agent-007"},
		{"empty", "   ", "agent"},
		{"special only", "!@#$%^", "agent"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SlugID(tt.input); got != tt.want {
				t.Errorf("SlugID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSanitizeEinoAgentID(t *testing.T) {
	tests := []struct {
		name string
		input string
		want  string
	}{
		{"normal", "CyberStrike-Deep", "cyberstrike-deep"},
		{"extra chars", "CyberStrike_Deep!", "cyberstrikedeep"},
		{"empty", "  ", "cyberstrike-deep"},
		{"already clean", "test-agent-123", "test-agent-123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeEinoAgentID(tt.input); got != tt.want {
				t.Errorf("sanitizeEinoAgentID() = %v, want %v", got, tt.want)
			}
		})
	}
}
