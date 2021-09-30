package slackhook

import (
	"fmt"
	"strings"

	"github.com/downballot/downballot/internal/httpextra"
	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

// Hook is a logrus hook for sending messages to Slack.
type Hook struct {
	slackChannel string
	slackClient  *slack.Client
	levels       []logrus.Level
}

// LevelToPrefixMap maps a log level to a text prefix.
var LevelToPrefixMap = map[logrus.Level]string{
	logrus.PanicLevel: ":rotating_light: ",
	logrus.FatalLevel: ":skull: ",
	logrus.ErrorLevel: ":x: ",
	logrus.WarnLevel:  ":warning: ",
	logrus.InfoLevel:  ":information_source: ",
	logrus.DebugLevel: "Debug: ",
	logrus.TraceLevel: "Trace: ",
}

// New returns a new Hook.
func New(slackClient *slack.Client, slackChannel string, minimumLevel logrus.Level) *Hook {
	h := &Hook{
		slackChannel: slackChannel,
		slackClient:  slackClient,
	}
	// Add each level to the list, and stop whenever we reach the minimum level.
	levels := []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
		logrus.TraceLevel,
	}
	for _, level := range levels {
		h.levels = append(h.levels, level)
		if level == minimumLevel {
			break
		}
	}
	return h
}

// Levels are the log levels to support.
func (h *Hook) Levels() []logrus.Level {
	return h.levels
}

// Fire a message.
func (h *Hook) Fire(entry *logrus.Entry) error {
	var blocks []slack.Block

	// If we know the logging trace information, then add that as a context block.
	if entry.Context != nil {
		var lines []string
		if value := entry.Context.Value(httpextra.ContextKeyHost); value != nil {
			lines = append(lines, fmt.Sprintf("Host: %s", value))
		}
		if value := entry.Context.Value(httpextra.ContextKeyPath); value != nil {
			lines = append(lines, fmt.Sprintf("Path: %s", value))
		}
		if len(lines) > 0 {
			emoji := true
			verbatim := false
			block := slack.NewContextBlock("", slack.NewTextBlockObject(slack.PlainTextType, strings.Join(lines, "\n"), emoji, verbatim))
			blocks = append(blocks, block)
		}
	}

	// Add a section with an emoji icon before the message.
	{
		message := LevelToPrefixMap[entry.Level] + entry.Message
		emoji := true
		verbatim := false
		block := slack.NewSectionBlock(slack.NewTextBlockObject(slack.PlainTextType, message, emoji, verbatim), nil, nil)
		blocks = append(blocks, block)
	}

	// Post the message.  The message-text option is needed for notifications to work properly.
	// The blocks are the modern way to format a message.
	_ /*messageID*/, _ /*messageTimestamp*/, err := h.slackClient.PostMessage(h.slackChannel, slack.MsgOptionText(entry.Message, false), slack.MsgOptionBlocks(blocks...))
	return err
}
