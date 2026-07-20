package cmd

import (
	"encoding/base64"
	"testing"

	"github.com/effectivai/riskos-cli/internal/api"
)

func TestDecodeStepContent(t *testing.T) {
	const markdown = "Generate keys in **Developer Workbench**. See [docs](https://help.socure.com)."
	encoded := base64.StdEncoding.EncodeToString([]byte(markdown))

	tests := []struct {
		name string
		step api.ChecklistStep
		want string
	}{
		{
			name: "markdown decodes base64",
			step: api.ChecklistStep{ContentType: "markdown", ContentSecondary: encoded},
			want: markdown,
		},
		{
			name: "markdown case-insensitive",
			step: api.ChecklistStep{ContentType: "Markdown", ContentSecondary: encoded},
			want: markdown,
		},
		{
			name: "plain text passes through",
			step: api.ChecklistStep{ContentType: "plaintext", ContentSecondary: "just plain text"},
			want: "just plain text",
		},
		{
			name: "empty content type passes through",
			step: api.ChecklistStep{ContentType: "", ContentSecondary: "raw content"},
			want: "raw content",
		},
		{
			name: "invalid base64 falls back to raw",
			step: api.ChecklistStep{ContentType: "markdown", ContentSecondary: "not-valid-base64!!!"},
			want: "not-valid-base64!!!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decodeStepContent(tt.step); got != tt.want {
				t.Errorf("decodeStepContent() = %q, want %q", got, tt.want)
			}
		})
	}
}
