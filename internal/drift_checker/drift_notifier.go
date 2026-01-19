package drift_checker

//counterfeiter:generate -o ./fakes/ . DriftNotifierInterface
type DriftNotifierInterface interface {
	Notify(summary DriftSummary)
}

type DriftSummary struct {
	NumPagesCompared  int
	NumDriftsDetected int
}
