package crawler

import (
	"mirrorer/internal/config"
	"mirrorer/internal/file"
	"net/http"

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
		colly.Async(true),
	)

	err := c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: cfg.Concurrency})

	if err != nil {
		return nil, err
	}

	c.OnRequest(func(r *colly.Request) {
		for header, value := range cfg.Headers {
			r.Headers.Set(header, value)
		}
	})

	// Set up a crawling logic
	c.OnHTML("a[href], link[href], img[src], script[src]", htmlHandler)

	// Save HTML redirects
	c.SetRedirectHandler(redirectHandler)

	// Save successful responses to disk
	c.OnResponse(responseHandler)

	// Handle errors
	c.OnError(errorHandler)

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

	err := e.Request.Visit(e.Request.AbsoluteURL(link))
	if err != nil {
		log.Debug().Err(err).Msg("Error attempting to visit link")
	}
}

func responseHandler(r *colly.Response) {
	contentType := r.Headers.Get("Content-Type")

	err := file.Save(r.Request.URL, contentType, r.Body)

	if err != nil {
		log.Error().Err(err).Msg("Error saving response to disk")
	}
}

func errorHandler(r *colly.Response, err error) {
	log.Error().Str("url", r.Request.URL.String()).Int("status", r.StatusCode).Err(err).Msg("Error returned from request")
}
