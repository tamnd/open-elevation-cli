// Package openelevation exposes the Open Elevation API as a kit Domain driver.
// A multi-domain host (ant) enables it with a single blank import:
//
//	import _ "github.com/tamnd/open-elevation-cli/open-elevation"
//
// The same Domain also builds the standalone elevation binary (see cli/root.go),
// so the binary and a host share one source of truth.
package openelevation

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

func init() { kit.Register(Domain{}) }

// Domain is the open-elevation driver.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "openelevation",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "elevation",
			Short:  "Elevation data for any GPS coordinate",
			Long: `elevation fetches GPS elevation data from api.open-elevation.com.
No API key required.`,
			Site: Host,
			Repo: "https://github.com/tamnd/open-elevation-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{
		Name:    "lookup",
		Group:   "read",
		Single:  true,
		Summary: "Fetch elevation for a single GPS coordinate",
	}, lookupOp)

	kit.Handle(app, kit.OpMeta{
		Name:    "batch",
		Group:   "read",
		List:    true,
		Summary: "Fetch elevation for multiple GPS coordinates",
	}, batchOp)
}

// newClient builds the client from host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- inputs ---

type lookupInput struct {
	Lat    float64 `kit:"flag,inherit" help:"latitude"`
	Lon    float64 `kit:"flag,inherit" help:"longitude"`
	Client *Client `kit:"inject"`
}

type batchInput struct {
	Locations string  `kit:"flag,inherit" help:"space-separated lat,lon pairs e.g. \"41.16,-8.58 10,10\""`
	Client    *Client `kit:"inject"`
}

// --- handlers ---

func lookupOp(ctx context.Context, in lookupInput, emit func(*Point) error) error {
	pt, err := in.Client.Lookup(ctx, in.Lat, in.Lon)
	if err != nil {
		return err
	}
	return emit(pt)
}

func batchOp(ctx context.Context, in batchInput, emit func(*Point) error) error {
	coords, err := parseCoords(in.Locations)
	if err != nil {
		return errs.Usage("invalid locations: %v", err)
	}
	pts, err := in.Client.Batch(ctx, coords)
	if err != nil {
		return err
	}
	for i := range pts {
		if err := emit(&pts[i]); err != nil {
			return err
		}
	}
	return nil
}

// parseCoords parses space-separated "lat,lon" pairs into [][2]float64.
func parseCoords(s string) ([][2]float64, error) {
	var out [][2]float64
	for _, pair := range strings.Fields(s) {
		parts := strings.SplitN(pair, ",", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("expected lat,lon got %q", pair)
		}
		lat, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return nil, fmt.Errorf("bad lat %q: %w", parts[0], err)
		}
		lon, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return nil, fmt.Errorf("bad lon %q: %w", parts[1], err)
		}
		out = append(out, [2]float64{lat, lon})
	}
	return out, nil
}

// --- Resolver ---

// Classify turns "lat,lon" input into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	if input == "" {
		return "", "", errs.Usage("empty open-elevation reference")
	}
	_, parseErr := parseCoords(input)
	if parseErr != nil {
		return "", "", errs.Usage("unrecognized open-elevation reference: %q", input)
	}
	return "point", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "point":
		return fmt.Sprintf("https://api.open-elevation.com/api/v1/lookup?locations=%s", id), nil
	default:
		return "", errs.Usage("openelevation has no resource type %q", uriType)
	}
}
