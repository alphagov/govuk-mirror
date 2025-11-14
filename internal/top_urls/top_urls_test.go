package top_urls_test

import (
	"fmt"
	"math/rand"
	"net/url"
	"testing"

	"mirrorer/internal/top_urls"

	"github.com/stretchr/testify/assert"
)

func TestTopUrls(t *testing.T) {
	urls := [10]string{
		"/Lorem",
		"/ipsum",
		"/dolor",
		"/sit",
		"/amet",
		"/consectetur",
		"/adipiscing",
		"/elit",
		"/sed",
		"/do",
	}

	counts := [10]int64{
		2_173,
		1_928,
		0,
		4_619,
		5,
		10,
		3,
		10_000,
		500,
		7_782,
	}

	urlHitCounts := make([]top_urls.UrlHitCount, len(urls))

	for i, u := range urls {
		parsedUrl, err := url.Parse(u)
		if err != nil {
			panic(fmt.Sprintf("Test setup url %s couldn't be parsed", u))
		}

		urlHitCounts[i] = top_urls.UrlHitCount{
			ViewedUrl: *parsedUrl,
			ViewCount: counts[i],
		}
	}

	t.Run("NewTopUrls with too many unsampled pages requested", func(t *testing.T) {
		topUrls, err := top_urls.NewTopUrls(urlHitCounts, 20, 5, rand.New(rand.NewSource(99)))
		assert.Nil(t, topUrls)
		assert.Error(t, err)
	})

	t.Run("NewTopUrls with too many sampled pages requested", func(t *testing.T) {
		topUrls, err := top_urls.NewTopUrls(urlHitCounts, 5, 6, rand.New(rand.NewSource(99)))
		assert.Nil(t, topUrls)
		assert.Error(t, err)
	})

	t.Run("NewTopUrls with top 3 unsampled and 2 sampled", func(t *testing.T) {
		expectedUnsampledUrls := []top_urls.UrlHitCount{
			urlHitCounts[7],
			urlHitCounts[9],
			urlHitCounts[3],
		}
		expectedSampledUrls := []top_urls.UrlHitCount{
			urlHitCounts[0],
			urlHitCounts[4],
		}

		topUrls, err := top_urls.NewTopUrls(urlHitCounts, 3, 2, rand.New(rand.NewSource(99)))

		assert.NoError(t, err)
		assert.Equal(t, expectedUnsampledUrls, topUrls.TopUnsampledUrls)
		assert.Equal(t, expectedSampledUrls, topUrls.RemainingSampledUrls)
	})

	t.Run("NewTopUrls with top 7 unsampled and 3 sampled", func(t *testing.T) {
		expectedUnsampledUrls := []top_urls.UrlHitCount{
			urlHitCounts[7],
			urlHitCounts[9],
			urlHitCounts[3],
			urlHitCounts[0],
			urlHitCounts[1],
			urlHitCounts[8],
			urlHitCounts[5],
		}
		expectedSampledUrls := []top_urls.UrlHitCount{
			urlHitCounts[2],
			urlHitCounts[6],
			urlHitCounts[4],
		}

		topUrls, err := top_urls.NewTopUrls(urlHitCounts, 7, 3, rand.New(rand.NewSource(99)))

		assert.NoError(t, err)
		assert.Equal(t, expectedUnsampledUrls, topUrls.TopUnsampledUrls)
		assert.Equal(t, expectedSampledUrls, topUrls.RemainingSampledUrls)
	})
}
