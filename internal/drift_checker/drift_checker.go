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
	log.Info().Msgf("Comparing top %d unsampled paths", len(urls.TopUnsampledUrls))
	topDriftsDetected, topComparisonsPerformed := comparePages(urls.TopUnsampledUrls, fetcher, comparer)

	log.Info().Msgf("Comparing remaining %d sampled paths", len(urls.RemainingSampledUrls))
	restDriftsDetected, restComparisonsPerformed := comparePages(urls.RemainingSampledUrls, fetcher, comparer)

	driftsDetected := topDriftsDetected + restDriftsDetected
	if driftsDetected > 0 {
		summary := DriftSummary{
			NumDriftsDetected: driftsDetected,
			NumPagesCompared:  topComparisonsPerformed + restComparisonsPerformed,
		}
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
// Returns the number of drifts that were detected among the URLs,
// and the total number of comparisons performed
func comparePages(
	pages []top_urls.UrlHitCount,
	fetcher page_fetcher.PageFetcherInterface,
	comparer page_comparer.PageComparerInterface,
) (int, int) {
	drifts := 0
	comparisons := 0

	for _, page := range pages {
		url := page.ViewedUrl.String()

		live, err := fetcher.FetchLivePage(url)
		if err != nil {
			log.Error().Err(err).Msgf("error fetching live page: %s", url)
			continue
		}

		mirror, err := fetcher.FetchMirrorPage(url)
		if err != nil {
			log.Error().Err(err).Msgf("error fetching mirror page: %s", url)
			continue
		}

		same, err := comparer.HaveSameBody(live, mirror)
		comparisons++
		if err != nil {
			log.Error().Err(err).Msgf("error comparing live and mirror pages: %s", url)
			continue
		}

		if !same {
			log.Info().Err(fmt.Errorf("drift detected between live and mirror on %s", url))
			drifts++
		}

		log.Info().Bool("drift", !same).Msgf("Comparing %s (%d views)", url, page.ViewCount)
	}

	return drifts, comparisons
}
