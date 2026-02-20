package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/saadjs/kcal-cli/internal/provider/openfoodfacts"
	"github.com/saadjs/kcal-cli/internal/provider/upcitemdb"
	"github.com/saadjs/kcal-cli/internal/provider/usda"
)

const defaultProviderSearchTTL = 7 * 24 * time.Hour

type FoodSearchResult struct {
	Provider              string             `json:"provider"`
	Description           string             `json:"description"`
	Brand                 string             `json:"brand"`
	ServingAmount         float64            `json:"serving_amount"`
	ServingUnit           string             `json:"serving_unit"`
	Calories              float64            `json:"calories"`
	ProteinG              float64            `json:"protein_g"`
	CarbsG                float64            `json:"carbs_g"`
	FatG                  float64            `json:"fat_g"`
	FiberG                float64            `json:"fiber_g"`
	SugarG                float64            `json:"sugar_g"`
	SodiumMg              float64            `json:"sodium_mg"`
	Micronutrients        Micronutrients     `json:"micronutrients,omitempty"`
	SourceID              int64              `json:"source_id"`
	SourceTier            string             `json:"source_tier,omitempty"`
	ConfidenceScore       float64            `json:"confidence_score,omitempty"`
	IsVerified            bool               `json:"is_verified,omitempty"`
	VerificationReasons   []string           `json:"verification_reasons,omitempty"`
	ProviderConfidence    float64            `json:"provider_confidence,omitempty"`
	NutritionCompleteness string             `json:"nutrition_completeness,omitempty"`
	Alternatives          []FoodSearchResult `json:"alternatives,omitempty"`
}

type FoodSearchOptions struct {
	Provider         string
	APIKey           string
	APIKeyType       string
	Limit            int
	VerifiedMinScore float64
	VerifiedOnly     bool
}

type FoodSearchCandidate struct {
	Provider string
	Options  BarcodeLookupOptions
}

type ProviderSearchCacheItem struct {
	Provider       string    `json:"provider"`
	Query          string    `json:"query"`
	LimitRequested int       `json:"limit_requested"`
	FetchedAt      time.Time `json:"fetched_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type searchClient interface {
	SearchFoods(ctx context.Context, query string, limit int) ([]BarcodeLookupResult, []byte, error)
}

func SearchFoods(db *sql.DB, query string, opts FoodSearchOptions) ([]FoodSearchResult, error) {
	provider := normalizeBarcodeProvider(opts.Provider)
	if provider == "" {
		provider = BarcodeProviderUSDA
	}
	results, err := searchFoodsByProvider(db, provider, query, opts)
	if err != nil {
		return nil, err
	}
	results = dedupeAndRankFoodSearch(results, []string{provider})
	if opts.VerifiedOnly {
		results = filterVerified(results)
	}
	return results, nil
}

func SearchFoodsWithFallback(db *sql.DB, query string, candidates []FoodSearchCandidate, opts FoodSearchOptions) ([]FoodSearchResult, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no lookup providers configured")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}
	providerOrder := make([]string, 0, len(candidates))
	all := make([]FoodSearchResult, 0, 16)
	for _, c := range candidates {
		provider := normalizeBarcodeProvider(c.Provider)
		if provider == "" {
			continue
		}
		providerOrder = append(providerOrder, provider)
		providerOpts := opts
		providerOpts.Provider = provider
		providerOpts.APIKey = c.Options.APIKey
		providerOpts.APIKeyType = c.Options.APIKeyType
		items, err := searchFoodsByProvider(db, provider, query, providerOpts)
		if err != nil {
			continue
		}
		all = append(all, items...)
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("search failed for %q across providers", query)
	}
	all = dedupeAndRankFoodSearch(all, providerOrder)
	if opts.VerifiedOnly {
		all = filterVerified(all)
	}
	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}
	return all, nil
}

func searchFoodsByProvider(db *sql.DB, provider, query string, opts FoodSearchOptions) ([]FoodSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}
	if opts.Limit <= 0 {
		opts.Limit = 10
	}
	if opts.Limit > 50 {
		opts.Limit = 50
	}
	if opts.VerifiedMinScore <= 0 {
		opts.VerifiedMinScore = DefaultVerifiedMinScore
	}

	if cached, found, err := lookupProviderSearchCache(db, provider, query, opts.Limit); err != nil {
		return nil, err
	} else if found {
		results := make([]FoodSearchResult, 0, len(cached))
		for _, item := range cached {
			item.Provider = provider
			item.SourceTier = "cache"
			item.NutritionCompleteness = deriveNutritionCompleteness(item)
			applySearchConfidence(&item, query, opts.VerifiedMinScore)
			results = append(results, toFoodSearchResult(item))
		}
		return results, nil
	}

	client, err := newSearchClient(provider, opts)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	items, raw, err := client.SearchFoods(ctx, query, opts.Limit)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Provider = provider
		items[i].SourceTier = "provider"
		items[i].NutritionCompleteness = deriveNutritionCompleteness(items[i])
		applySearchConfidence(&items[i], query, opts.VerifiedMinScore)
	}
	if err := upsertProviderSearchCache(db, provider, query, opts.Limit, items, raw, time.Now().Add(defaultProviderSearchTTL)); err != nil {
		return nil, err
	}

	out := make([]FoodSearchResult, 0, len(items))
	for _, item := range items {
		out = append(out, toFoodSearchResult(item))
	}
	return out, nil
}

func newSearchClient(provider string, opts FoodSearchOptions) (searchClient, error) {
	switch provider {
	case BarcodeProviderUSDA:
		return &usdaClientAdapter{client: &usda.Client{APIKey: opts.APIKey}}, nil
	case BarcodeProviderOpenFoodFacts:
		return &openFoodFactsClientAdapter{client: &openfoodfacts.Client{}}, nil
	case BarcodeProviderUPCItemDB:
		return &upcItemDBClientAdapter{client: &upcitemdb.Client{APIKey: opts.APIKey, APIKeyType: opts.APIKeyType}}, nil
	default:
		return nil, fmt.Errorf("unsupported provider %q", provider)
	}
}

func toFoodSearchResult(in BarcodeLookupResult) FoodSearchResult {
	return FoodSearchResult{
		Provider:              in.Provider,
		Description:           in.Description,
		Brand:                 in.Brand,
		ServingAmount:         in.ServingAmount,
		ServingUnit:           in.ServingUnit,
		Calories:              in.Calories,
		ProteinG:              in.ProteinG,
		CarbsG:                in.CarbsG,
		FatG:                  in.FatG,
		FiberG:                in.FiberG,
		SugarG:                in.SugarG,
		SodiumMg:              in.SodiumMg,
		Micronutrients:        in.Micronutrients,
		SourceID:              in.SourceID,
		SourceTier:            in.SourceTier,
		ConfidenceScore:       in.ConfidenceScore,
		IsVerified:            in.IsVerified,
		VerificationReasons:   in.VerificationReasons,
		ProviderConfidence:    in.ProviderConfidence,
		NutritionCompleteness: in.NutritionCompleteness,
	}
}

func dedupeAndRankFoodSearch(items []FoodSearchResult, providerOrder []string) []FoodSearchResult {
	if len(items) == 0 {
		return nil
	}
	groups := map[string][]FoodSearchResult{}
	for _, item := range items {
		key := canonicalSearchKey(item.Description, item.Brand)
		groups[key] = append(groups[key], item)
	}
	providerRank := map[string]int{}
	for i, p := range providerOrder {
		providerRank[p] = i
	}
	out := make([]FoodSearchResult, 0, len(groups))
	for _, group := range groups {
		sort.SliceStable(group, func(i, j int) bool {
			return compareFoodSearch(group[i], group[j], providerRank)
		})
		primary := group[0]
		if len(group) > 1 {
			primary.Alternatives = append(primary.Alternatives, group[1:]...)
		}
		out = append(out, primary)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return compareFoodSearch(out[i], out[j], providerRank)
	})
	return out
}

func filterVerified(items []FoodSearchResult) []FoodSearchResult {
	out := make([]FoodSearchResult, 0, len(items))
	for _, item := range items {
		if item.IsVerified {
			out = append(out, item)
		}
	}
	return out
}

func compareFoodSearch(a, b FoodSearchResult, providerRank map[string]int) bool {
	if a.ConfidenceScore != b.ConfidenceScore {
		return a.ConfidenceScore > b.ConfidenceScore
	}
	ca := completenessRank(a.NutritionCompleteness)
	cb := completenessRank(b.NutritionCompleteness)
	if ca != cb {
		return ca > cb
	}
	ra, oka := providerRank[a.Provider]
	rb, okb := providerRank[b.Provider]
	if oka && okb && ra != rb {
		return ra < rb
	}
	if oka != okb {
		return oka
	}
	return strings.ToLower(a.Description) < strings.ToLower(b.Description)
}

func completenessRank(v string) int {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "complete":
		return 2
	case "partial":
		return 1
	default:
		return 0
	}
}

func canonicalSearchKey(description, brand string) string {
	re := regexp.MustCompile(`[^a-z0-9]+`)
	normalize := func(s string) string {
		s = strings.ToLower(strings.TrimSpace(s))
		s = re.ReplaceAllString(s, " ")
		s = strings.Join(strings.Fields(s), " ")
		return s
	}
	return normalize(description) + "|" + normalize(brand)
}

func lookupProviderSearchCache(db *sql.DB, provider, query string, limit int) ([]BarcodeLookupResult, bool, error) {
	queryNorm := canonicalSearchKey(query, "")
	var raw string
	var expiresAtRaw string
	err := db.QueryRow(`
SELECT raw_json, expires_at
FROM provider_search_cache
WHERE provider = ? AND query_norm = ? AND limit_requested = ?
`, provider, queryNorm, limit).Scan(&raw, &expiresAtRaw)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("lookup provider search cache: %w", err)
	}
	expiresAt, err := time.Parse(time.RFC3339, expiresAtRaw)
	if err != nil {
		return nil, false, fmt.Errorf("parse provider search cache expiry: %w", err)
	}
	if time.Now().After(expiresAt) {
		return nil, false, nil
	}
	var items []BarcodeLookupResult
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, false, fmt.Errorf("decode provider search cache: %w", err)
	}
	return items, true, nil
}

func upsertProviderSearchCache(db *sql.DB, provider, query string, limit int, normalized []BarcodeLookupResult, _ []byte, expiresAt time.Time) error {
	query = strings.TrimSpace(query)
	queryNorm := canonicalSearchKey(query, "")
	payload, err := json.Marshal(normalized)
	if err != nil {
		return fmt.Errorf("marshal provider search cache payload: %w", err)
	}

	_, err = db.Exec(`
INSERT INTO provider_search_cache(provider, query, query_norm, limit_requested, raw_json, fetched_at, expires_at)
VALUES(?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider, query_norm, limit_requested) DO UPDATE SET
  query=excluded.query,
  raw_json=excluded.raw_json,
  fetched_at=excluded.fetched_at,
  expires_at=excluded.expires_at
`, provider, query, queryNorm, limit, string(payload), time.Now().Format(time.RFC3339), expiresAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("upsert provider search cache: %w", err)
	}
	return nil
}

func applySearchConfidence(result *BarcodeLookupResult, query string, minScore float64) {
	if result == nil {
		return
	}
	confidence := ScoreSearchConfidence(*result, query, minScore)
	result.ConfidenceScore = confidence.Score
	result.ProviderConfidence = confidence.Score
	result.IsVerified = confidence.IsVerified
	result.VerificationReasons = confidence.Reasons
}

func ListProviderSearchCache(db *sql.DB, provider, query string, limit int) ([]ProviderSearchCacheItem, error) {
	provider = normalizeBarcodeProvider(provider)
	queryNorm := canonicalSearchKey(query, "")
	if limit <= 0 {
		limit = 100
	}
	base := `SELECT provider, query, limit_requested, fetched_at, expires_at FROM provider_search_cache`
	args := make([]any, 0, 3)
	clauses := make([]string, 0, 2)
	if provider != "" {
		clauses = append(clauses, "provider = ?")
		args = append(args, provider)
	}
	if strings.TrimSpace(query) != "" {
		clauses = append(clauses, "query_norm = ?")
		args = append(args, queryNorm)
	}
	if len(clauses) > 0 {
		base += " WHERE " + strings.Join(clauses, " AND ")
	}
	base += " ORDER BY fetched_at DESC LIMIT ?"
	args = append(args, limit)
	rows, err := db.Query(base, args...)
	if err != nil {
		return nil, fmt.Errorf("list provider search cache: %w", err)
	}
	defer rows.Close()
	out := make([]ProviderSearchCacheItem, 0)
	for rows.Next() {
		var item ProviderSearchCacheItem
		var fetched, expires string
		if err := rows.Scan(&item.Provider, &item.Query, &item.LimitRequested, &fetched, &expires); err != nil {
			return nil, fmt.Errorf("scan provider search cache: %w", err)
		}
		item.FetchedAt, _ = time.Parse(time.RFC3339, fetched)
		item.ExpiresAt, _ = time.Parse(time.RFC3339, expires)
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider search cache: %w", err)
	}
	return out, nil
}

func PurgeProviderSearchCache(db *sql.DB, provider, query string, purgeAll bool) (int64, error) {
	provider = normalizeBarcodeProvider(provider)
	queryNorm := canonicalSearchKey(query, "")
	var (
		res sql.Result
		err error
	)
	switch {
	case purgeAll:
		res, err = db.Exec(`DELETE FROM provider_search_cache`)
	case provider != "" && strings.TrimSpace(query) != "":
		res, err = db.Exec(`DELETE FROM provider_search_cache WHERE provider = ? AND query_norm = ?`, provider, queryNorm)
	case provider != "":
		res, err = db.Exec(`DELETE FROM provider_search_cache WHERE provider = ?`, provider)
	case strings.TrimSpace(query) != "":
		res, err = db.Exec(`DELETE FROM provider_search_cache WHERE query_norm = ?`, queryNorm)
	default:
		return 0, fmt.Errorf("specify --all, --provider, --query, or provider+query")
	}
	if err != nil {
		return 0, fmt.Errorf("purge provider search cache: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("provider search cache rows affected: %w", err)
	}
	return affected, nil
}
