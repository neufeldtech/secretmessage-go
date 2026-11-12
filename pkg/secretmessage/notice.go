package secretmessage

import (
	"fmt"
	"os"
	"strings"
)

// DefaultShutdownNoticeMessage is the default notice message informing users of the hosted service shutdown.
const DefaultShutdownNoticeMessage = "The hosted service (secretmessage.xyz Slack app) will be shutting down after Dec 31st 2026 and this repo will be archived at that time as well."

// GetShutdownNotice returns the formatted shutdown notice string if configured, or empty string if not.
// It checks ctl.config.ShutdownNotice, falling back to os.Getenv("SHUTDOWN_NOTICE"),
// and ctl.config.SurveyLink, falling back to os.Getenv("SURVEY_LINK").
func (ctl *PublicController) GetShutdownNotice() string {
	notice := ctl.config.ShutdownNotice
	if notice == "" {
		notice = os.Getenv("SHUTDOWN_NOTICE")
	}

	link := ctl.config.SurveyLink
	if link == "" {
		link = os.Getenv("SURVEY_LINK")
	}

	return FormatShutdownNotice(notice, link)
}

// FormatShutdownNotice formats the shutdown notice message according to the provided notice text and link.
func FormatShutdownNotice(notice, link string) string {
	notice = strings.TrimSpace(notice)
	link = strings.TrimSpace(link)

	// Explicitly disabled
	switch strings.ToLower(notice) {
	case "false", "0", "off", "disable", "disabled":
		return ""
	}

	defaultNotice := fmt.Sprintf(":warning: %s", DefaultShutdownNoticeMessage)

	if notice != "" {
		switch {
		case strings.EqualFold(notice, "true") || notice == "1":
			if link != "" {
				return fmt.Sprintf("_<%s|%s>_", link, defaultNotice)
			}
			return fmt.Sprintf("_%s_", defaultNotice)
		case strings.HasPrefix(notice, "http://") || strings.HasPrefix(notice, "https://"):
			return fmt.Sprintf("_<%s|%s>_", notice, defaultNotice)
		default:
			msg := notice
			if !strings.HasPrefix(msg, ":warning:") && !strings.HasPrefix(msg, "⚠️") {
				msg = fmt.Sprintf(":warning: %s", msg)
			}
			if link != "" && !strings.Contains(notice, "<http") {
				return fmt.Sprintf("_<%s|%s>_", link, msg)
			}
			return fmt.Sprintf("_%s_", msg)
		}
	}

	if link != "" {
		if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
			return fmt.Sprintf("_<%s|%s>_", link, defaultNotice)
		}
		msg := link
		if !strings.HasPrefix(msg, ":warning:") && !strings.HasPrefix(msg, "⚠️") {
			msg = fmt.Sprintf(":warning: %s", msg)
		}
		return fmt.Sprintf("_%s_", msg)
	}

	return ""
}
