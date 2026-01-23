package drift_checker_test

import (
	"errors"
	"mirrorer/internal/drift_checker"
	notifier_fakes "mirrorer/internal/drift_checker/fakes"
	page_comparer_fakes "mirrorer/internal/page_comparer/fakes"
	"mirrorer/internal/page_fetcher"
	page_fetcher_fakes "mirrorer/internal/page_fetcher/fakes"
	"mirrorer/internal/top_urls"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func asUrl(str string) url.URL {
	u, _ := url.Parse(str)
	return *u
}

func htmlPage(body string) *page_fetcher.Page {
	return &page_fetcher.Page{
		Body:        body,
		ContentType: "text/html",
	}
}

func TestDriftChecker(t *testing.T) {

	t.Run("fetches the live and mirror versions of each page", func(t *testing.T) {
		urls := &top_urls.TopUrls{
			TopUnsampledUrls: []top_urls.UrlHitCount{
				{
					ViewedUrl: asUrl("/page-1"),
					ViewCount: 100,
				},
			},
			RemainingSampledUrls: []top_urls.UrlHitCount{
				{
					ViewedUrl: asUrl("/page-2"),
					ViewCount: 10,
				},
			},
		}

		fetcher := page_fetcher_fakes.FakePageFetcherInterface{}
		comparer := page_comparer_fakes.FakePageComparerInterface{}

		fetcher.FetchLivePageReturns(htmlPage("str"), nil)
		fetcher.FetchMirrorPageReturns(htmlPage("str"), nil)
		comparer.HaveSameBodyReturns(true, nil)

		drift_checker.CheckPagesForDrift(urls, &fetcher, &comparer, &notifier_fakes.FakeDriftNotifierInterface{})

		assert.Equal(t, 2, fetcher.FetchLivePageCallCount())
		assert.Equal(t, 2, fetcher.FetchMirrorPageCallCount())

		livePageUrls := []string{}
		mirrorPageUrls := []string{}

		for i := 0; i < 2; i++ {
			liveUrl := fetcher.FetchLivePageArgsForCall(i)
			mirrorUrl := fetcher.FetchMirrorPageArgsForCall(i)

			livePageUrls = append(livePageUrls, liveUrl)
			mirrorPageUrls = append(mirrorPageUrls, mirrorUrl)
		}

		assert.Equal(t, []string{"/page-1", "/page-2"}, livePageUrls)
		assert.Equal(t, []string{"/page-1", "/page-2"}, mirrorPageUrls)
	})

	t.Run("compares the bodies of each live and mirror pair", func(t *testing.T) {
		urls := &top_urls.TopUrls{
			TopUnsampledUrls: []top_urls.UrlHitCount{
				{
					ViewedUrl: asUrl("/page-1"),
					ViewCount: 100,
				},
			},
			RemainingSampledUrls: []top_urls.UrlHitCount{
				{
					ViewedUrl: asUrl("/page-2"),
					ViewCount: 10,
				},
			},
		}

		fetcher := page_fetcher_fakes.FakePageFetcherInterface{}
		comparer := page_comparer_fakes.FakePageComparerInterface{}

		liveFetcherStub := func(path string) (*page_fetcher.Page, error) {
			return htmlPage("LIVE-" + path), nil
		}
		mirrorFetcherStub := func(path string) (*page_fetcher.Page, error) {
			return htmlPage("MIRROR-" + path), nil
		}

		fetcher.FetchLivePageCalls(liveFetcherStub)
		fetcher.FetchMirrorPageCalls(mirrorFetcherStub)
		comparer.HaveSameBodyReturns(true, nil)

		drift_checker.CheckPagesForDrift(urls, &fetcher, &comparer, &notifier_fakes.FakeDriftNotifierInterface{})

		assert.Equal(t, 2, comparer.HaveSameBodyCallCount())

		callPairs := [][]string{}
		expected := [][]string{
			{"LIVE-/page-1", "MIRROR-/page-1"},
			{"LIVE-/page-2", "MIRROR-/page-2"},
		}

		for i := 0; i < 2; i++ {
			a, b := comparer.HaveSameBodyArgsForCall(i)

			callPairs = append(callPairs, []string{a.Body, b.Body})
		}

		assert.Equal(t, expected, callPairs)
	})

	t.Run("if there are no drifts, it does not send an alert, and returns false to indicate no drifts", func(t *testing.T) {
		urls := &top_urls.TopUrls{
			TopUnsampledUrls: []top_urls.UrlHitCount{
				{
					ViewedUrl: asUrl("/page-1"),
					ViewCount: 100,
				},
			},
			RemainingSampledUrls: []top_urls.UrlHitCount{
				{
					ViewedUrl: asUrl("/page-2"),
					ViewCount: 10,
				},
			},
		}

		fetcher := page_fetcher_fakes.FakePageFetcherInterface{}
		comparer := page_comparer_fakes.FakePageComparerInterface{}
		notifier := notifier_fakes.FakeDriftNotifierInterface{}

		fetcher.FetchLivePageReturns(htmlPage("str"), nil)
		fetcher.FetchMirrorPageReturns(htmlPage("str"), nil)
		comparer.HaveSameBodyReturns(true, nil)

		drifts := drift_checker.CheckPagesForDrift(urls, &fetcher, &comparer, &notifier)

		assert.False(t, drifts)
		assert.Equal(t, 0, notifier.NotifyCallCount())
	})

	t.Run("if there are any drifts, it sends an alert with a summary of the findings, and returns true to indicate >0 drifts were found", func(t *testing.T) {
		urls := &top_urls.TopUrls{
			TopUnsampledUrls: []top_urls.UrlHitCount{
				{
					ViewedUrl: asUrl("/page-1"),
					ViewCount: 100,
				},
			},
			RemainingSampledUrls: []top_urls.UrlHitCount{
				{
					ViewedUrl: asUrl("/page-2"),
					ViewCount: 10,
				},
			},
		}

		fetcher := page_fetcher_fakes.FakePageFetcherInterface{}
		comparer := page_comparer_fakes.FakePageComparerInterface{}
		notifier := notifier_fakes.FakeDriftNotifierInterface{}

		fetcher.FetchLivePageReturns(htmlPage("str"), nil)
		fetcher.FetchMirrorPageReturns(htmlPage("str"), nil)

		comparer.HaveSameBodyReturnsOnCall(0, false, nil)
		comparer.HaveSameBodyReturnsOnCall(1, true, nil)

		drifts := drift_checker.CheckPagesForDrift(urls, &fetcher, &comparer, &notifier)

		assert.True(t, drifts)
		assert.Equal(t, 1, notifier.NotifyCallCount())
		summary := notifier.NotifyArgsForCall(0)
		assert.Equal(t, 2, summary.NumPagesCompared)
		assert.Equal(t, 1, summary.NumDriftsDetected)
	})

	t.Run("if there are any drifts, the summary includes the number of errors encountered during comparisons", func(t *testing.T) {
		urls := &top_urls.TopUrls{
			TopUnsampledUrls: []top_urls.UrlHitCount{
				{
					ViewedUrl: asUrl("/page-1"),
					ViewCount: 100,
				},
			},
			RemainingSampledUrls: []top_urls.UrlHitCount{
				{
					ViewedUrl: asUrl("/page-2"),
					ViewCount: 10,
				},
				{
					ViewedUrl: asUrl("/page-3"),
					ViewCount: 5,
				},
			},
		}

		fetcher := page_fetcher_fakes.FakePageFetcherInterface{}
		comparer := page_comparer_fakes.FakePageComparerInterface{}
		notifier := notifier_fakes.FakeDriftNotifierInterface{}

		fetcher.FetchLivePageReturns(htmlPage("str"), nil)
		fetcher.FetchMirrorPageReturns(htmlPage("str"), nil)

		comparer.HaveSameBodyReturnsOnCall(0, false, nil)
		comparer.HaveSameBodyReturnsOnCall(1, true, nil)
		comparer.HaveSameBodyReturnsOnCall(2, false, errors.New("failed to compare"))

		_ = drift_checker.CheckPagesForDrift(urls, &fetcher, &comparer, &notifier)

		assert.Equal(t, 1, notifier.NotifyCallCount())
		summary := notifier.NotifyArgsForCall(0)
		assert.Equal(t, 1, summary.NumErrors)
	})
}
