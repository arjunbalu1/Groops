package services

import (
	"groops/internal/database"
	"groops/internal/models"
	"log"
	"strings"

	"gorm.io/gorm"
)

type SearchResult struct {
	Group models.Group `json:"group"`
	Score float64      `json:"score"`
	Rank  float64      `json:"rank"`
}

type SearchService struct {
	db *gorm.DB
}

func NewSearchService() *SearchService {
	return &SearchService{
		db: database.GetDB(),
	}
}

// SearchGroups performs advanced search with ranking and fuzzy matching
func (s *SearchService) SearchGroups(searchTerm string, limit int, offset int) ([]models.Group, error) {
	if strings.TrimSpace(searchTerm) == "" {
		return []models.Group{}, nil
	}

	// Clean and prepare search term
	cleanTerm := strings.TrimSpace(searchTerm)

	// Multi-strategy search results
	var results []SearchResult

	// Strategy 1: Full-Text Search with ranking (highest priority)
	ftsResults, err := s.fullTextSearch(cleanTerm, limit)
	if err != nil {
		log.Printf("FTS search error: %v", err)
	} else {
		results = append(results, ftsResults...)
	}

	// Strategy 2: Fuzzy matching for typos (medium priority)
	fuzzyResults, err := s.fuzzySearch(cleanTerm)
	if err != nil {
		log.Printf("Fuzzy search error: %v", err)
	} else {
		results = append(results, fuzzyResults...)
	}

	// Strategy 3: Partial matching fallback (lowest priority)
	partialResults, err := s.partialSearch(cleanTerm)
	if err != nil {
		log.Printf("Partial search error: %v", err)
	} else {
		results = append(results, partialResults...)
	}

	// Combine and deduplicate results
	combinedResults := s.combineAndRankResults(results)

	// Apply pagination
	start := offset
	end := offset + limit
	if start >= len(combinedResults) {
		return []models.Group{}, nil
	}
	if end > len(combinedResults) {
		end = len(combinedResults)
	}

	// Extract groups from results
	var groups []models.Group
	for i := start; i < end; i++ {
		groups = append(groups, combinedResults[i].Group)
	}

	return groups, nil
}

// fullTextSearch performs PostgreSQL full-text search
func (s *SearchService) fullTextSearch(searchTerm string, limit int) ([]SearchResult, error) {
	// Clean and prepare search term for tsquery
	cleanTerm := strings.TrimSpace(searchTerm)
	if cleanTerm == "" {
		return []SearchResult{}, nil
	}

	// Use the sophisticated search query preparation
	tsqueryTerm := s.prepareSearchQuery(cleanTerm)
	if tsqueryTerm == "" {
		return []SearchResult{}, nil
	}

	var results []SearchResult

	query := `
		SELECT *, 
		       ts_rank_cd(search_vector, to_tsquery('english', ?), 1) as fts_rank
		FROM "group" 
		WHERE search_vector @@ to_tsquery('english', ?)
		  AND date_time > NOW()
		ORDER BY fts_rank DESC
		LIMIT ?
	`

	rows, err := s.db.Raw(query, tsqueryTerm, tsqueryTerm, limit).Rows()
	if err != nil {
		log.Printf("FTS search error: %v", err)
		return []SearchResult{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var group models.Group
		var rank float64
		var searchVector interface{} // For the search_vector column

		// Scan all group fields plus search_vector and the rank
		err := rows.Scan(
			&group.ID, &group.Name, &group.DateTime, &group.Location,
			&group.Cost, &group.SkillLevel, &group.ActivityType, &group.MaxMembers,
			&group.Description, &group.OrganiserID, &group.CreatedAt, &group.UpdatedAt,
			&searchVector, // Add this for the search_vector column
			&rank,
		)
		if err != nil {
			log.Printf("Error scanning FTS result: %v", err)
			continue
		}

		results = append(results, SearchResult{
			Group: group,
			Score: rank * 100, // High priority for FTS
			Rank:  rank,
		})
	}

	return results, nil
}

// fuzzySearch performs fuzzy matching using pg_trgm for typos
func (s *SearchService) fuzzySearch(searchTerm string) ([]SearchResult, error) {
	var results []SearchResult

	query := `
		SELECT id, name, date_time, location, cost, skill_level, activity_type, 
		       max_members, description, organiser_id, created_at, updated_at,
			   GREATEST(
				   similarity(name, $1),
				   similarity(activity_type, $1),
				   similarity(description, $1)
			   ) as fuzzy_score
		FROM "group" 
		WHERE (
			   name % $1 OR 
			   activity_type % $1 OR 
			   description % $1
		   )
		   AND date_time > NOW()
		   AND GREATEST(
			   similarity(name, $1),
			   similarity(activity_type, $1),
			   similarity(description, $1)
		   ) > 0.3
		ORDER BY fuzzy_score DESC
		LIMIT 30
	`

	rows, err := s.db.Raw(query, searchTerm).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var group models.Group
		var similarity float64

		// Scan all group fields plus the similarity score
		err := rows.Scan(
			&group.ID, &group.Name, &group.DateTime, &group.Location,
			&group.Cost, &group.SkillLevel, &group.ActivityType, &group.MaxMembers,
			&group.Description, &group.OrganiserID, &group.CreatedAt, &group.UpdatedAt,
			&similarity,
		)
		if err != nil {
			log.Printf("Error scanning fuzzy result: %v", err)
			continue
		}

		results = append(results, SearchResult{
			Group: group,
			Score: similarity * 50, // Medium priority for fuzzy
			Rank:  similarity,
		})
	}

	return results, nil
}

// partialSearch performs partial matching as fallback
func (s *SearchService) partialSearch(searchTerm string) ([]SearchResult, error) {
	var results []SearchResult

	searchPattern := "%" + strings.ToLower(searchTerm) + "%"

	query := `
		SELECT id, name, date_time, location, cost, skill_level, activity_type, 
		       max_members, description, organiser_id, created_at, updated_at,
			   CASE 
				   WHEN LOWER(name) LIKE $1 THEN 3
				   WHEN LOWER(activity_type) LIKE $1 THEN 2
				   WHEN LOWER(description) LIKE $1 THEN 1
				   ELSE 0.5
			   END as partial_score
		FROM "group" 
		WHERE (
			   LOWER(name) LIKE $1 OR 
			   LOWER(activity_type) LIKE $1 OR 
			   LOWER(description) LIKE $1 OR
			   LOWER(organiser_id) LIKE $1
		   )
		   AND date_time > NOW()
		ORDER BY partial_score DESC
		LIMIT 20
	`

	rows, err := s.db.Raw(query, searchPattern).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var group models.Group
		var score float64

		// Scan all group fields plus the partial score
		err := rows.Scan(
			&group.ID, &group.Name, &group.DateTime, &group.Location,
			&group.Cost, &group.SkillLevel, &group.ActivityType, &group.MaxMembers,
			&group.Description, &group.OrganiserID, &group.CreatedAt, &group.UpdatedAt,
			&score,
		)
		if err != nil {
			log.Printf("Error scanning partial result: %v", err)
			continue
		}

		results = append(results, SearchResult{
			Group: group,
			Score: score * 10, // Low priority for partial
			Rank:  score,
		})
	}

	return results, nil
}

// prepareSearchQuery converts user input to tsquery format
func (s *SearchService) prepareSearchQuery(searchTerm string) string {
	// Clean and split terms
	terms := strings.Fields(strings.ToLower(searchTerm))
	if len(terms) == 0 {
		return ""
	}

	// Handle single word
	if len(terms) == 1 {
		return terms[0] + ":*" // Prefix matching
	}

	// Handle multiple words - use OR logic for broader, more user-friendly results
	processedTerms := make([]string, len(terms))
	for i, term := range terms {
		processedTerms[i] = term + ":*"
	}

	return strings.Join(processedTerms, " | ") // OR logic for better coverage
}

// combineAndRankResults merges results from different strategies and removes duplicates
func (s *SearchService) combineAndRankResults(results []SearchResult) []SearchResult {
	// Group by group ID and take the best score
	groupMap := make(map[string]SearchResult)

	for _, result := range results {
		existing, exists := groupMap[result.Group.ID]
		if !exists || result.Score > existing.Score {
			groupMap[result.Group.ID] = result
		}
	}

	// Convert back to slice and sort by score
	var finalResults []SearchResult
	for _, result := range groupMap {
		finalResults = append(finalResults, result)
	}

	// Sort by score descending
	for i := 0; i < len(finalResults)-1; i++ {
		for j := i + 1; j < len(finalResults); j++ {
			if finalResults[i].Score < finalResults[j].Score {
				finalResults[i], finalResults[j] = finalResults[j], finalResults[i]
			}
		}
	}

	return finalResults
}
