package infrastructure

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lividlabs/codefall/cli/internal/shared/version"
)

// ReleasesURL is where the release workflow publishes each release, the same host the install script
// reads.
const ReleasesURL = "https://install.codefall.dev"

// maxDownload bounds one response body. A release archive is a few megabytes.
const maxDownload = 256 << 20

// downloadTimeout bounds one request, end to end, so a stalled connection fails instead of hanging.
const downloadTimeout = 5 * time.Minute

// HTTPReleases reads release files over HTTPS.
type HTTPReleases struct {
	client  *http.Client
	baseURL string
}

// NewHTTPReleases builds the releases gateway over baseURL, which tests point at a local server.
func NewHTTPReleases(baseURL string) *HTTPReleases {
	return &HTTPReleases{
		client:  &http.Client{Timeout: downloadTimeout},
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

// Latest is the text of the `latest` file, trimmed of whitespace.
func (r *HTTPReleases) Latest(ctx context.Context) (string, error) {
	body, err := r.get(ctx, r.baseURL+"/latest")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}

// Fetch is one file from a release's folder, which is named by the version without a leading v.
func (r *HTTPReleases) Fetch(ctx context.Context, release version.Version, name string) ([]byte, error) {
	return r.get(ctx, r.baseURL+"/"+release.String()+"/"+name)
}

func (r *HTTPReleases) get(ctx context.Context, url string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	response, err := r.client.Do(request)
	if err != nil {
		return nil, err
	}

	defer func() { _ = response.Body.Close() }()

	// The bucket answers a file it does not hold with 403, not 404.
	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%s is not published (%s)", url, response.Status)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, response.Status)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxDownload+1))
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}

	if len(body) > maxDownload {
		return nil, fmt.Errorf("GET %s: response is larger than %d bytes", url, maxDownload)
	}

	return body, nil
}
