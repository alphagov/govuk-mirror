package crawler

import (
	"errors"
	"mime"
	"mirrorer/internal/client"
	"mirrorer/internal/config"
	"mirrorer/internal/file"
	"net/http"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/rs/zerolog/log"
)

type Crawler struct {
	cfg       *config.Config
	collector *colly.Collector
}

func NewCrawler(cfg *config.Config) (*Crawler, error) {
	collector, err := newCollector(cfg)
	if err != nil {
		return nil, err
	}

	return &Crawler{cfg: cfg, collector: collector}, nil
}

func newCollector(cfg *config.Config) (*colly.Collector, error) {
	c := colly.NewCollector(
		colly.UserAgent(cfg.UserAgent),
		colly.AllowedDomains(cfg.AllowedDomains...),
		colly.URLFilters(cfg.URLFilters...),
		colly.DisallowedURLFilters(cfg.DisallowedURLFilters...),
		colly.Async(true),
	)

	client := client.NewClient(c, redirectHandler)
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
	c.OnError(errorHandler)

	// Save successful responses to disk
	c.OnResponse(responseHandler)

	// Set up a crawling logic
	c.OnHTML("a[href], link[href], img[src], script[src]", htmlHandler)

	// Crawl sitemap indexes and sitemaps
	c.OnXML("//sitemapindex/sitemap/loc", xmlHandler)
	c.OnXML("//urlset/url/loc", xmlHandler)

	return c, nil
}

func (cr *Crawler) Run() {
	// Start the crawler
	err := cr.collector.Visit(cr.cfg.Site)
	if err != nil {
		log.Fatal().Err(err).Msg("Error starting the crawler")
	}

	cr.collector.Wait()
}

func redirectHandler(req *http.Request, via []*http.Request) error {
	for _, redirectReq := range via {
		body := file.RedirectHTMLBody(req.URL.String())
		err := file.Save(redirectReq.URL, "text/html", body)
		if err != nil {
			return err
		}
	}
	return nil
}

func htmlHandler(e *colly.HTMLElement) {
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

	err := e.Request.Visit(link)
	if err != nil && !isForbiddenURLError(err) {
		log.Error().Err(err).Str("link", link).Msg("Error attempting to visit link")
	}
}

func xmlHandler(e *colly.XMLElement) {
	err := e.Request.Visit(e.Text)
	if err != nil && !isForbiddenURLError(err) {
		log.Error().Err(err).Str("link", e.Text).Msg("Error attempting to visit link")
	}
}

func responseHandler(r *colly.Response) {
	contentType := r.Headers.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		log.Error().Err(err).Str("crawled_url", r.Request.URL.String()).Msg("Error parsing Content-Type header")
	}
	if mediaType == "text/css" {
		urls := file.FindCssUrls(r.Body)

		for _, url := range urls {
			err := r.Request.Visit(url)
			if err != nil && !isForbiddenURLError(err) {
				log.Error().Err(err).Str("link", url).Msg("Error attempting to visit link")
			}
		}
	} else if strings.Contains(mediaType, "openxmlformats") || strings.Contains(mediaType, "+xml") {
		// This is a hacky work around colly's handleOnXML behaviour which
		// considers any response body with content-type containing the
		// substring "xml" to be parsed as XML. This is an incorrect assumption
		// for docx, xlsx, pptx files which aren't strictly xml structured and
		// cause parsing errors. This also stops unnessary parsing of
		// non-sitemap files (e.g. svg or rdf).
		r.Headers.Set("Content-Type", strings.ReplaceAll(contentType, "xml", ""))
	}

	err = file.Save(r.Request.URL, contentType, r.Body)
	if err != nil {
		log.Error().Err(err).Str("crawled_url", r.Request.URL.String()).Msg("Error saving response to disk")
	} else {
		log.Info().Str("crawled_url", r.Request.URL.String()).Str("type", mediaType).Msg("Downloaded file")
	}
}

func errorHandler(r *colly.Response, err error) {
	if errors.Is(err, client.DisallowedURLError{}) {
		// Normal behaviour to not follow the URL, so we can just ignore this error
		return
	}

	log.Error().Err(err).Int("status", r.StatusCode).Str("crawled_url", r.Request.URL.String()).Msg("Error returned from request")
}

func isForbiddenURLError(err error) bool {
	return errors.Is(err, colly.ErrForbiddenDomain) || errors.Is(err, colly.ErrForbiddenURL) || errors.As(err, new(*colly.AlreadyVisitedError))
}
