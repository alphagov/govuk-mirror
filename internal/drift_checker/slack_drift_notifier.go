package drift_checker

import (
	"fmt"

	"github.com/slack-go/slack"
)

type SlackDriftNotifier struct {
	client    *slack.Client
	channelId string
}

func NewSlackDriftNotifier(client *slack.Client, channelId string) *SlackDriftNotifier {
	return &SlackDriftNotifier{
		client:    client,
		channelId: channelId,
	}
}

func (s SlackDriftNotifier) Notify(summary DriftSummary) error {
	txt := fmt.Sprintf(`
	Drifts were detected beween the live and mirror versions of pages on GOV.UK
	Pages tested: %d
	Drifts detected: %d

	Look at the logs in Logit to find out more
	`,
		summary.NumPagesCompared,
		summary.NumDriftsDetected,
	)
	msg := slack.MsgOptionText(txt, false)

	_, _, err := s.client.PostMessage(s.channelId, msg)
	return err
}
