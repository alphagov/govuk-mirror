package crawler

import (
	"errors"
	"mime"
	"mirrorer/internal/client"
	"mirrorer/internal/config"
	"mirrorer/internal/file"
	"net/http"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/queue"
	"github.com/rs/zerolog/log"
)

type Crawler struct {
	cfg       *config.Config
	collector *colly.Collector
	queue     *queue.Queue
}

func NewCrawler(cfg *config.Config) (*Crawler, error) {
	q, _ := queue.New(
		cfg.Concurrency, // Number of consumer threads
		&queue.InMemoryQueueStorage{MaxSize: 10000000}, // Use default queue storage
	)

	collector, err := newCollector(cfg, q)
	if err != nil {
		return nil, err
	}

	return &Crawler{cfg: cfg, collector: collector, queue: q}, nil
}

func newCollector(cfg *config.Config, q *queue.Queue) (*colly.Collector, error) {
	c := colly.NewCollector(
		colly.UserAgent(cfg.UserAgent),
		colly.AllowedDomains(cfg.AllowedDomains...),
		colly.DisallowedURLFilters(cfg.DisallowedURLFilters...),
		//colly.Async(true),
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

	// Set up a crawling logic
	c.OnHTML("a[href], link[href], img[src], script[src]", func(e *colly.HTMLElement) {
		var link string
		switch e.Name {
		case "a", "link":
			link = e.Attr("href")
		case "img", "script":
			link = e.Attr("src")
		}

		err := q.AddURL(e.Request.AbsoluteURL(link))
		if err != nil && !isForbiddenURLError(err) {
			log.Error().Err(err).Msg("Error attempting to visit link")
		}
	})

	xmlHandler := func(e *colly.XMLElement) {
		err := q.AddURL(e.Request.AbsoluteURL(e.Text))
		if err != nil && !isForbiddenURLError(err) {
			log.Error().Err(err).Msg("Error attempting to visit link")
		}
	}

	// Crawl sitemap indexes and sitemaps
	c.OnXML("//sitemapindex/sitemap/loc", xmlHandler)
	c.OnXML("//urlset/url/loc", xmlHandler)

	// Save successful responses to disk
	c.OnResponse(responseHandler)

	// Handle errors
	c.OnError(errorHandler)

	return c, nil
}

func (cr *Crawler) Run() {
	err := cr.queue.AddURL(cr.cfg.Site)
	if err != nil {
		log.Fatal().Err(err).Msg("Error queuing initial URL")
	}

	err = cr.queue.Run(cr.collector)
	if err != nil {
		log.Fatal().Err(err).Msg("Error starting the crawler")
	}
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

func responseHandler(r *colly.Response) {
	contentType := r.Headers.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		log.Error().Str("url", r.Request.URL.String()).Err(err).Msg("Error parsing Content-Type header")
	}
	if mediaType == "text/css" {
		urls := file.FindCssUrls(r.Body)

		for _, url := range urls {
			err := r.Request.Visit(r.Request.AbsoluteURL(url))
			if err != nil && !isForbiddenURLError(err) {
				log.Error().Err(err).Msg("Error attempting to visit link")
			}
		}
	}

	err = file.Save(r.Request.URL, contentType, r.Body)

	if err != nil {
		log.Error().Str("url", r.Request.URL.String()).Err(err).Msg("Error saving response to disk")
	}
}

func errorHandler(r *colly.Response, err error) {
	if errors.Is(err, client.DisallowedURLError{}) {
		// Normal behaviour to not follow the URL, so we can just ignore this error
		return
	}
	log.Error().Str("url", r.Request.URL.String()).Int("status", r.StatusCode).Err(err).Msg("Error returned from request")
}

func isForbiddenURLError(err error) bool {
	return errors.Is(err, colly.ErrForbiddenDomain) || errors.Is(err, colly.ErrForbiddenURL) || errors.Is(err, colly.ErrAlreadyVisited)
}
