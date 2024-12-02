# GOV.UK Mirror

A concurrent crawler and site downloader to make a local copy of a website.
This is used by GOV.UK to populate mirrors hosted by AWS S3 and GCP Storage.

## Usage

Configuration is handled through environment variables as listed below:

- SITE: Specifies the starting URL for the crawler.
    - Example: `SITE=https://www.gov.uk`
- ALLOWED_DOMAINS: A comma-separated list of hostnames permitted to be crawled.
    - Example: `ALLOWED_DOMAINS=domain1.com,domain2.com`
- USER_AGENT: Customizes the user agent for requests. Defaults to `govukbot` if not specified.
    - Example: `USER_AGENT=custom-user-agent`
- HEADERS: Provides custom headers for requests.
    - Example: `HEADERS=Rate-Limit-Token:ABC123,X-Header:X-Value`
- CONCURRENCY: Controls the number of concurrent requests, useful for controlling request rate.
    - Example: `CONCURRENCY=10`
- URL_RULES: A comma-separated list of regex patterns matching URLs that the crawler should crawl. All other URLs will be avoided.
    - Example: `URL_RULES=https://www.gov.uk/.*`
- DISALLOWED_URL_RULES: A comma-separated list of regex patterns matching URLs that the crawler should avoid.
    - Example: `DISALLOWED_URL_RULES=/search/.*,/government/.*\.atom`
