package top_urls

import (
	"fmt"
	"math/rand"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

var urls = [10]string{
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

var counts = [10]int64{
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

var urlHitCounts = make([]UrlHitCount, len(urls))

func TestMain(m *testing.M) {
	for i, u := range urls {
		parsedUrl, err := url.Parse(u)
		if err != nil {
			panic(fmt.Sprintf("Test setup url %s couldn't be parsed", u))
		}

		urlHitCounts[i] = UrlHitCount{
			viewedUrl: *parsedUrl,
			viewCount: counts[i],
		}
	}

	m.Run()
}

func TestNewTopUrlsWithTooManyUnsampledRequested(t *testing.T) {
	topUrls, err := NewTopUrls(urlHitCounts, 20, 5, rand.New(rand.NewSource(99)))
	assert.Nil(t, topUrls)
	assert.Error(t, err)
}

func TestNewTopUrlsWithTooManySampledRequested(t *testing.T) {
	topUrls, err := NewTopUrls(urlHitCounts, 5, 6, rand.New(rand.NewSource(99)))
	assert.Nil(t, topUrls)
	assert.Error(t, err)
}

func TestNewTopUrlsWithTop3And2Sampled(t *testing.T) {
	expectedUnsampledUrls := []UrlHitCount{
		urlHitCounts[7],
		urlHitCounts[9],
		urlHitCounts[3],
	}
	expectedSampledUrls := []UrlHitCount{
		urlHitCounts[0],
		urlHitCounts[4],
	}

	topUrls, err := NewTopUrls(urlHitCounts, 3, 2, rand.New(rand.NewSource(99)))

	assert.NoError(t, err)
	assert.Equal(t, expectedUnsampledUrls, topUrls.topUnsampledUrls)
	assert.Equal(t, expectedSampledUrls, topUrls.remainingSampledUrls)
}

func TestNewTopUrlsWithTop7And3Sampled(t *testing.T) {
	expectedUnsampledUrls := []UrlHitCount{
		urlHitCounts[7],
		urlHitCounts[9],
		urlHitCounts[3],
		urlHitCounts[0],
		urlHitCounts[1],
		urlHitCounts[8],
		urlHitCounts[5],
	}
	expectedSampledUrls := []UrlHitCount{
		urlHitCounts[2],
		urlHitCounts[6],
		urlHitCounts[4],
	}

	topUrls, err := NewTopUrls(urlHitCounts, 7, 3, rand.New(rand.NewSource(99)))

	assert.NoError(t, err)
	assert.Equal(t, expectedUnsampledUrls, topUrls.topUnsampledUrls)
	assert.Equal(t, expectedSampledUrls, topUrls.remainingSampledUrls)
}
