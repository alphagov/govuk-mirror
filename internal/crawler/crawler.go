package crawler

import (
	"mirrorer/internal/config"

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
	c := colly.NewCollector()

	// Set up a crawling logic
	c.OnHTML("a[href], link[href], img[src], script[src]", htmlHandler)

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

func errorHandler(r *colly.Response, err error) {
	log.Error().Str("url", r.Request.URL.String()).Int("status", r.StatusCode).Err(err).Msg("Error returned from request")
}
