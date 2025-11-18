package top_urls_test

import "mirrorer/internal/top_urls"

type MockTopUrlsClient struct {
	topUrls *top_urls.TopUrls
	err     error
}

func NewMockTopUrlsClient(topUrls *top_urls.TopUrls, err error) (top_urls.TopUrlsClientInterface, error) {
	return MockTopUrlsClient{
		topUrls: topUrls,
		err:     err,
	}, err
}

func (mock MockTopUrlsClient) GetTopUrls() (*top_urls.TopUrls, error) {
	if mock.err != nil {
		return nil, mock.err
	}

	return mock.topUrls, nil
}
