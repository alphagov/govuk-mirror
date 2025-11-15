package mirror_comparator

import (
	"mirrorer/internal/config"
	"mirrorer/internal/top_urls"
)

type MirrorComparator struct {
	comparisonConfig config.MirrorComparisonConfig
	topUrlsClient    top_urls.TopUrlsClientInterface
}

type MirrorComparisonResult struct {
	TotalPagesCompared       int
	PagesWithIdenticalBodies []PageComparison
	PagesWithDifferences     []PageComparison
	SuccessfulComparison     bool
}

type PageComparison struct {
	URL                string
	MirrorFetchSuccess bool
	OriginFetchSuccess bool
	MirrorFetchError   *error
	OriginFetchError   *error
	MirrorBodyChecksum string
	OriginBodyChecksum string
}

func NewMirrorComparator(cfg config.MirrorComparisonConfig, topUrlsClient top_urls.TopUrlsClientInterface) *MirrorComparator {
	return &MirrorComparator{
		comparisonConfig: cfg,
		topUrlsClient:    topUrlsClient,
	}
}

func (mc MirrorComparator) CompareMirrorToOrigin() (MirrorComparisonResult, error) {
	result := MirrorComparisonResult{
		TotalPagesCompared:       0,
		PagesWithIdenticalBodies: []PageComparison{},
		PagesWithDifferences:     []PageComparison{},
		SuccessfulComparison:     false,
	}

	return result, nil
}
