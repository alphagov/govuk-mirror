package drift_checker

import (
	"fmt"
	"mirrorer/internal/page_comparer"
	"mirrorer/internal/page_fetcher"
	"mirrorer/internal/top_urls"

	"github.com/rs/zerolog/log"
)

// CheckPagesForDrift iterates through the given URLs, fetches their live and
// mirror versions, and compares them for drift. If it finds any, it emits a
// notification with a summary.
//
// Returns true if any drifts were found
func CheckPagesForDrift(
	urls *top_urls.TopUrls,
	fetcher page_fetcher.PageFetcherInterface,
	comparer page_comparer.PageComparerInterface,
	notifier DriftNotifierInterface,
) bool {
	summary := DriftSummary{}

	log.Info().Msgf("Comparing top %d unsampled paths", len(urls.TopUnsampledUrls))
	comparePages(urls.TopUnsampledUrls, fetcher, comparer, &summary)

	log.Info().Msgf("Comparing remaining %d sampled paths", len(urls.RemainingSampledUrls))
	comparePages(urls.RemainingSampledUrls, fetcher, comparer, &summary)

	if summary.NumDriftsDetected > 0 {
		err := notifier.Notify(summary)
		if err != nil {
			log.Error().Err(err).Msgf("failed to send drift summary notification")
		}
		return true
	}
	return false
}

// comparePages runs through each of the URLs and compares their
// live and mirror versions.
//
// Increments the counts on the supplied summary struct
func comparePages(
	pages []top_urls.UrlHitCount,
	fetcher page_fetcher.PageFetcherInterface,
	comparer page_comparer.PageComparerInterface,
	summary *DriftSummary,
) {
	for _, page := range pages {
		url := page.ViewedUrl.String()

		live, err := fetcher.FetchLivePage(url)
		if err != nil {
			log.Error().Err(err).Msgf("error fetching live page: %s", url)
			summary.NumErrors++
			continue
		}

		mirror, err := fetcher.FetchMirrorPage(url)
		if err != nil {
			log.Error().Err(err).Msgf("error fetching mirror page: %s", url)
			summary.NumErrors++
			continue
		}

		same, err := comparer.HaveSameBody(*live, *mirror)
		summary.NumPagesCompared++
		if err != nil {
			log.Error().Err(err).Msgf("error comparing live and mirror pages: %s", url)
			summary.NumErrors++
			continue
		}

		if !same {
			log.Info().Err(fmt.Errorf("drift detected between live and mirror on %s", url))
			summary.NumDriftsDetected++
		}

		log.Info().
			Bool("drift", !same).
			Bool("live_body_empty", live.Body == "").
			Bool("mirror_body_empty", mirror.Body == "").
			Msgf("Comparing %s (%d views)", url, page.ViewCount)
	}
}
