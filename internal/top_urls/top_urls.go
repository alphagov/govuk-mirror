package top_urls

import (
	"cmp"
	"fmt"
	"math/rand"
	"net/url"
	"slices"
)

type TopUrls struct {
	TopUnsampledUrls     []UrlHitCount
	RemainingSampledUrls []UrlHitCount
}

type UrlHitCount struct {
	ViewedUrl url.URL
	ViewCount int64
}

func NewTopUrls(urls []UrlHitCount, numberUnsampled int, numberToSample int, random *rand.Rand) (*TopUrls, error) {
	totalUrls := len(urls)
	if totalUrls < numberUnsampled {
		return nil, fmt.Errorf("requested %d unsampled, but there are only %d topUrls", numberUnsampled, totalUrls)
	}

	numUrlsAvailableToSample := totalUrls - numberUnsampled
	if numUrlsAvailableToSample < numberToSample {
		return nil, fmt.Errorf(
			"requested %d sampled, but there are only %d urls available to sample once the top %d are removed",
			numberToSample,
			numUrlsAvailableToSample,
			numberUnsampled,
		)
	}

	copiedUrls := make([]UrlHitCount, totalUrls)

	numCopiedUrls := copy(copiedUrls, urls)
	if numCopiedUrls != totalUrls {
		return nil, fmt.Errorf("expected there to be %d urls to sample from, but there was only %d", totalUrls, numCopiedUrls)
	}

	slices.SortFunc(copiedUrls, func(a, b UrlHitCount) int {
		return cmp.Compare(b.ViewCount, a.ViewCount)
	})

	sampledUrls, err := sampleRemainingUrls(copiedUrls[numberUnsampled:totalUrls], numberToSample, random)
	if err != nil {
		return nil, err
	}

	return &TopUrls{
		TopUnsampledUrls:     copiedUrls[0:numberUnsampled],
		RemainingSampledUrls: sampledUrls,
	}, nil
}

func sampleRemainingUrls(urlsToSample []UrlHitCount, numberToSample int, random *rand.Rand) ([]UrlHitCount, error) {
	maxPossibleSamples := len(urlsToSample)
	if numberToSample > maxPossibleSamples {
		return nil, fmt.Errorf("cannot sample %d elements since there are only %d to choose from", numberToSample, maxPossibleSamples)
	}

	random.Shuffle(len(urlsToSample), func(i, j int) {
		urlsToSample[i], urlsToSample[j] = urlsToSample[j], urlsToSample[i]
	})

	return urlsToSample[0:numberToSample], nil
}
