package top_urls

type MockTopUrlsClient struct {
	topUrls *TopUrls
	err     error
}

func NewMockTopUrlsClient(topUrls *TopUrls, err error) (TopUrlsClientInterface, error) {
	return MockTopUrlsClient{
		topUrls: topUrls,
		err:     err,
	}, err
}

func (mock MockTopUrlsClient) GetTopUrls() (*TopUrls, error) {
	if mock.err != nil {
		return nil, mock.err
	}

	return mock.topUrls, nil
}
