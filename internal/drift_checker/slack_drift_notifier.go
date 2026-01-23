package drift_checker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

type SlackDriftNotifier struct {
	webhookUrl url.URL
}

func NewSlackDriftNotifier(webhookUrl url.URL) *SlackDriftNotifier {
	return &SlackDriftNotifier{
		webhookUrl: webhookUrl,
	}
}

func (s SlackDriftNotifier) Notify(summary DriftSummary) error {
	txt := fmt.Sprintf(`
	Drifts were detected beween the live and mirror versions of pages on GOV.UK
	Pages tested: %d
	Drifts detected: %d
	Errors encountered: %d

	Look at the logs in Logit to find out more.
	Search "kubernetes.labels.app_kubernetes_io\/name: mirror-drift-check"
	`,
		summary.NumPagesCompared,
		summary.NumDriftsDetected,
		summary.NumErrors,
	)

	client := &http.Client{}
	jsonFields := map[string]interface{}{
		"text": txt,
	}
	body, err := json.Marshal(jsonFields)
	if err != nil {
		return err
	}

	resp, err := client.Post(s.webhookUrl.String(), "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	defer (func() {
		_ = resp.Body.Close()
	})()

	if resp.StatusCode != http.StatusOK {
		return errors.New("unexpected status code: " + resp.Status)
	}

	return nil
}
