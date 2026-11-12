package secretmessage_test

import (
	"os"
	"testing"

	"github.com/neufeldtech/secretmessage-go/pkg/secretmessage"
	"github.com/stretchr/testify/assert"
)

func TestFormatShutdownNotice(t *testing.T) {
	tests := []struct {
		name     string
		notice   string
		link     string
		expected string
	}{
		{
			name:     "both empty",
			notice:   "",
			link:     "",
			expected: "",
		},
		{
			name:     "explicitly disabled with false",
			notice:   "false",
			link:     "https://secretmessage.xyz",
			expected: "",
		},
		{
			name:     "explicitly disabled with 0",
			notice:   "0",
			link:     "https://secretmessage.xyz",
			expected: "",
		},
		{
			name:     "explicitly disabled with off",
			notice:   "off",
			link:     "https://secretmessage.xyz",
			expected: "",
		},
		{
			name:     "only survey link provided as URL",
			notice:   "",
			link:     "https://secretmessage.xyz",
			expected: "_<https://secretmessage.xyz|:warning: " + secretmessage.DefaultShutdownNoticeMessage + ">_",
		},
		{
			name:     "only survey link provided as text",
			notice:   "",
			link:     "Service shutting down soon",
			expected: "_:warning: Service shutting down soon_",
		},
		{
			name:     "notice is boolean true with survey link",
			notice:   "true",
			link:     "https://secretmessage.xyz",
			expected: "_<https://secretmessage.xyz|:warning: " + secretmessage.DefaultShutdownNoticeMessage + ">_",
		},
		{
			name:     "notice is boolean true without link",
			notice:   "true",
			link:     "",
			expected: "_:warning: " + secretmessage.DefaultShutdownNoticeMessage + "_",
		},
		{
			name:     "notice is URL",
			notice:   "https://secretmessage.xyz/shutdown",
			link:     "",
			expected: "_<https://secretmessage.xyz/shutdown|:warning: " + secretmessage.DefaultShutdownNoticeMessage + ">_",
		},
		{
			name:     "custom notice text without link",
			notice:   "Custom shutdown announcement",
			link:     "",
			expected: "_:warning: Custom shutdown announcement_",
		},
		{
			name:     "custom notice text with warning emoji already present",
			notice:   ":warning: Custom shutdown announcement",
			link:     "",
			expected: "_:warning: Custom shutdown announcement_",
		},
		{
			name:     "custom notice text with link",
			notice:   "Custom shutdown announcement",
			link:     "https://secretmessage.xyz",
			expected: "_<https://secretmessage.xyz|:warning: Custom shutdown announcement>_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := secretmessage.FormatShutdownNotice(tt.notice, tt.link)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetShutdownNotice_FromConfig(t *testing.T) {
	ctl := secretmessage.NewController(
		secretmessage.Config{
			SurveyLink: "https://secretmessage.xyz",
		},
		nil,
		nil,
	)

	notice := ctl.GetShutdownNotice()
	assert.Contains(t, notice, "https://secretmessage.xyz")
	assert.Contains(t, notice, secretmessage.DefaultShutdownNoticeMessage)
}

func TestGetShutdownNotice_FromEnv(t *testing.T) {
	os.Setenv("SURVEY_LINK", "https://secretmessage.xyz")
	defer os.Unsetenv("SURVEY_LINK")

	ctl := secretmessage.NewController(
		secretmessage.Config{},
		nil,
		nil,
	)

	notice := ctl.GetShutdownNotice()
	assert.Contains(t, notice, "https://secretmessage.xyz")
	assert.Contains(t, notice, secretmessage.DefaultShutdownNoticeMessage)
}
