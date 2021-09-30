package slackhook

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slacktest"
	"github.com/stretchr/testify/require"
)

func TestHook(t *testing.T) {
	slackServer := slacktest.NewTestServer()
	slackServer.Start()
	defer slackServer.Stop()
	logrus.Infof("Slack API URL: %s", slackServer.GetAPIURL())

	slackToken := "slack-abc123"
	slackChannel := "general"
	minimumSlackLogLevel := logrus.WarnLevel
	debugSlack := false // Set this to true to debug what Slack is doing.

	for _, goodClient := range []bool{false, true} {
		t.Run(fmt.Sprintf("goodClient=%t", goodClient), func(t *testing.T) {
			slackURL := slackServer.GetAPIURL()
			if !goodClient {
				slackURL += "xxx"
			}
			slackClient := slack.New(slackToken, slack.OptionDebug(debugSlack), slack.OptionAPIURL(slackURL))

			logger, _ := logrustest.NewNullLogger()

			hook := New(slackClient, slackChannel, minimumSlackLogLevel)
			logger.AddHook(hook)

			// This will log to the #general channel.
			rows := []struct {
				level   logrus.Level
				success bool
			}{
				{
					level:   logrus.DebugLevel,
					success: true, // We'll never log this in either case.
				},
				{
					level:   logrus.InfoLevel,
					success: true, // We'll never log this in either case.
				},
				{
					level:   logrus.WarnLevel,
					success: goodClient, // If the slack client is good, we'll successfully write to the channel; if not, this will fail.
				},
				{
					level:   logrus.ErrorLevel,
					success: goodClient, // If the slack client is good, we'll successfully write to the channel; if not, this will fail.
				},
			}
			for rowIndex, row := range rows {
				t.Run(fmt.Sprintf("%d/%s", rowIndex, row.level.String()), func(t *testing.T) {
					entry := logrus.NewEntry(logger)
					entry.Message = "test message"
					err := logger.Hooks.Fire(row.level, entry)
					if !row.success {
						require.NotNil(t, err, "Expected an error")
						return
					}
					require.Nil(t, err, "Expected no error")
				})
			}
		})
	}
}
