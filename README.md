# GOV.UK Mirror

A concurrent crawler and site downloader to make a local copy of a website.
This is used by GOV.UK to populate mirrors hosted by AWS S3 and GCP Storage.

## Usage

Configuration is handled through environment variables as listed below:

| Variable | Example | Description |
|----------|---------|-------------|
| `SITE` | `https://www-origin.publishing.service.gov.uk` | Specifies the starting URL for the crawler. |
| `ALLOWED_DOMAINS` | `domain1.com,domain2.com` | A comma-separated list of hostnames permitted to be crawled. |
| `USER_AGENT` | `custom-user-agent` | Customizes the user agent for requests. Defaults to `govuk-mirror-bot` if not specified. |
| `HEADERS` | `Rate-Limit-Token:ABC123,X-Header:X-Value` | Provides custom headers for requests. |
| `CONCURRENCY` | `10` | Controls the number of concurrent requests, useful for controlling request rate. |
| `URL_RULES` | `https://www-origin.publishing.service.gov.uk/.*` | A comma-separated list of regex patterns matching URLs that the crawler should crawl. All other URLs will be avoided. |
| `DISALLOWED_URL_RULES` | `/search/.*,/government/.*\.atom` | A comma-separated list of regex patterns matching URLs that the crawler should avoid. |
| `SKIP_VALIDATION` | `true` | Skip domain accessibility validation before crawling. Useful for offline testing. |
| `ASYNC` | `true` | Async crawling. Set to false for testing as a race condition could fail the crawler tests. |

## Crawling order

The crawler will scrape the most recent sites first according to the `lastmod` in the sitemap for their URL. In some cases where the `lastmod` is missing this value will be set to `2000-01-01` which means that it will be scraped at the end of the job.

## Metrics

Mirror pushes the following metrics to Prometheus Pushgateway:

| Metric | Description  |
|----------|----------------------|
| `crawled_pages_total` | Total number of HTTP errors encountered by the crawler |
| `crawler_errors_total` | Total number of HTTP errors encountered by the crawler |
| `download_errors_total` | Total number of download errors encountered by the crawler |
| `download_total` | Total number of files downloaded by the crawler |

### View metrics locally

1. Start up Prometheus Pushgateway

```
docker run -d -p 9091:9091 prom/pushgateway
```

2. Start up the mirror locally

```
make test-local
```

## How to deploy

This needs manual deployment to staging and production. Once the `Release` GitHub Action has run select the `Run workflow` 
option from the `Deploy` GitHub action. Then enter the latest tag number and the environment to deploy to.