package drift_checker

import "fmt"

type StdOutDriftNotifier struct{}

func (s StdOutDriftNotifier) Notify(summary DriftSummary) error {
	fmt.Printf(`
	Drifts were detected beween the live and mirror versions of pages on GOV.UK
	Pages tested: %d
	Drifts detected: %d
	Errors encountered: %d`,
		summary.NumPagesCompared,
		summary.NumDriftsDetected,
		summary.NumErrors,
	)

	return nil
}
