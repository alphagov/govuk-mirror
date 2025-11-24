package crawler

import (
	"errors"
	"mime"
	"mirrorer/internal/client"
	"mirrorer/internal/config"
	"mirrorer/internal/file"
	"mirrorer/internal/metrics"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/antchfx/xmlquery"
	"github.com/gocolly/colly/v2"

	"github.com/rs/zerolog/log"
)

type entry struct {
	val string
	key string
}

type CrawlState struct {
	entries         []entry
	numSitemaps     int
	counterSitemaps int
	isScraping      bool
}

type Crawler struct {
	cfg       *config.Config
	collector *colly.Collector
}

func NewCrawler(cfg *config.Config, m *metrics.Metrics) (*Crawler, error) {
	collector, err := newCollector(cfg, m)
	if err != nil {
		return nil, err
	}

	return &Crawler{cfg: cfg, collector: collector}, nil
}

func newCollector(cfg *config.Config, m *metrics.Metrics) (*colly.Collector, error) {
	c := colly.NewCollector(
		colly.UserAgent(cfg.UserAgent),
		colly.AllowedDomains(cfg.AllowedDomains...),
		colly.URLFilters(cfg.URLFilters...),
		colly.DisallowedURLFilters(cfg.DisallowedURLFilters...),
		colly.Async(cfg.Async),
	)

	crawlState := &CrawlState{
		entries:         []entry{},
		numSitemaps:     0,
		counterSitemaps: 0,
		isScraping:      false,
	}

	client := client.NewClient(c, redirectHandler(m))
	c.SetClient(client)

	err := c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: cfg.Concurrency})
	if err != nil {
		return nil, err
	}

	c.OnRequest(func(r *colly.Request) {
		for header, value := range cfg.Headers {
			r.Headers.Set(header, value)
		}
	})

	// Handle errors
	c.OnError(errorHandler(m))

	// Save successful responses to disk
	c.OnResponse(responseHandler(m))

	// Set up a crawling logic
	c.OnHTML("a[href], link[href], img[src], script[src]", htmlHandler())

	// crawl sitemap index
	c.OnXML("//sitemapindex", sitemapXmlHandler(crawlState))
	// crawl urlset in sitemap
	c.OnXML("//urlset", urlsetXmlHandler(crawlState))

	c.OnScraped(scrapeHandler(crawlState))

	return c, nil
}

func (cr *Crawler) Run(m *metrics.Metrics) {

	startTime := time.Now()

	defer metrics.CrawlerDuration(m, startTime)

	// Start the crawler
	err := cr.collector.Visit(cr.cfg.Site)
	if err != nil {
		log.Fatal().Err(err).Msg("Error starting the crawler")
	}

	cr.collector.Wait()
}

func redirectHandler(m *metrics.Metrics) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		for _, redirectReq := range via {
			body := file.RedirectHTMLBody(req.URL.String())
			metrics.CrawledPagesCounter(m)
			err := file.Save(redirectReq.URL, "text/html", body)
			metrics.DownloadCounter(m)
			if err != nil {
				metrics.DownloadCrawlerError(m)
				return err
			}
		}
		return nil
	}
}

func htmlHandler() func(e *colly.HTMLElement) {
	return func(e *colly.HTMLElement) {
		var link string
		switch e.Name {
		case "a", "link":
			link = e.Attr("href")
		case "img", "script":
			link = e.Attr("src")
		}

		if strings.HasPrefix(link, "#") {
			return
		}

		_ = e.Request.Visit(link)
	}
}

func sitemapXmlHandler(crawlState *CrawlState) func(e *colly.XMLElement) {
	return func(e *colly.XMLElement) {
		nodes, _ := xmlquery.QueryAll(e.DOM.(*xmlquery.Node), "//sitemap")
		crawlState.numSitemaps = len(nodes)

		xmlquery.FindEach(e.DOM.(*xmlquery.Node), "//sitemap", func(i int, child *xmlquery.Node) {
			_ = e.Request.Visit(child.SelectElement("loc").InnerText())
		})
	}
}

func urlsetXmlHandler(crawlState *CrawlState) func(e *colly.XMLElement) {
	return func(e *colly.XMLElement) {
		xmlquery.FindEach(e.DOM.(*xmlquery.Node), "//url", func(i int, child *xmlquery.Node) {
			var lastmod string
			if child.SelectElement("lastmod") == nil {
				log.Info().Str("loc", child.SelectElement("loc").InnerText()).Msg("No lastmod element")
				lastmod = "2000-01-01T00:00:00Z"
			} else {
				lastmod = child.SelectElement("lastmod").InnerText()
			}
			crawlState.entries = append(crawlState.entries, entry{
				val: lastmod,
				key: child.SelectElement("loc").InnerText()})
		})
		crawlState.counterSitemaps += 1
	}
}

func scrapeHandler(crawlState *CrawlState) func(*colly.Response) {
	return func(r *colly.Response) {
		if crawlState.isScraping || r.Request.URL.String() == "/sitemap.xml" || crawlState.counterSitemaps < crawlState.numSitemaps {
			return
		}
		crawlState.isScraping = true

		slices.SortFunc(crawlState.entries, func(a, b entry) int {
			return strings.Compare(a.val, b.val)
		})
		slices.Reverse(crawlState.entries)
		for _, ei := range crawlState.entries {
			_ = r.Request.Visit(ei.key)
		}
	}
}

func responseHandler(m *metrics.Metrics) func(*colly.Response) {
	return func(r *colly.Response) {

		contentType := r.Headers.Get("Content-Type")

		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil {
			metrics.HttpCrawlerError(m)
			log.Error().Err(err).Str("crawled_url", r.Request.URL.String()).Msg("Error parsing Content-Type header")
		}
		if mediaType == "text/css" {
			urls := file.FindCssUrls(r.Body)

			for _, url := range urls {
				_ = r.Request.Visit(url)
			}
		} else if strings.Contains(mediaType, "openxmlformats") || strings.Contains(mediaType, "+xml") {
			/*
				Some responses are in the Office OpenXML format (e.g. docx, xlsx, pptx) which aren't
				strictly XML structured and have in their Content-Type header "xml" as a substring. Parsing
				such responses as XML causes errors.

				This hacky workaround involves stripping "xml" from the Content-Type header to prevent Colly
				from trying to parse these files as XML. This also stops unnecessary parsing of non-sitemap files
				(e.g. svg or rdf).
			*/

			r.Headers.Set("Content-Type", strings.ReplaceAll(contentType, "xml", ""))
		}

		metrics.CrawledPagesCounter(m)

		err = file.Save(r.Request.URL, contentType, r.Body)
		if err != nil {
			metrics.DownloadCrawlerError(m)
			log.Error().Err(err).Str("crawled_url", r.Request.URL.String()).Msg("Error saving response to disk")
		} else {
			metrics.DownloadCounter(m)
			log.Info().Str("crawled_url", r.Request.URL.String()).Str("type", mediaType).Msg("Downloaded file")
		}
	}
}

func isForbiddenURLError(err error) bool {
	return errors.Is(err, colly.ErrForbiddenDomain) || errors.Is(err, colly.ErrForbiddenURL) || errors.As(err, new(*colly.AlreadyVisitedError))
}

func errorHandler(m *metrics.Metrics) func(*colly.Response, error) {
	return func(r *colly.Response, err error) {
		if errors.Is(err, client.DisallowedURLError{}) || isForbiddenURLError(err) {
			// Normal behaviour to not follow the URL, so we can just ignore this error
			return
		}

		metrics.HttpCrawlerError(m)
		log.Error().Err(err).Int("status", r.StatusCode).Str("crawled_url", r.Request.URL.String()).Msg("Error returned from request")
	}
}
