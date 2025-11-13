package top_urls

type TopUrlsClientInterface interface {
	GetTopUrls() (*TopUrls, error)
}
