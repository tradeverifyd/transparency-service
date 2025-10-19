package generator

import (
	"fmt"
	"strings"
)

// GenerateIssuerURN creates an issuer URN for a company
// Format: urn:supply-chain:{company-slug}
func GenerateIssuerURN(companySlug string) string {
	return fmt.Sprintf("urn:supply-chain:%s", companySlug)
}

// GenerateSubjectURN creates a subject URN for a document
// Format: urn:supply-chain:{company-slug}:{type}:{id}
func GenerateSubjectURN(companySlug, docType, docID string) string {
	return fmt.Sprintf("urn:supply-chain:%s:%s:%s", companySlug, docType, docID)
}

// GenerateContentLocation creates an HTTPS URL for a document
// Format: https://{company-slug}.example/supply-chain/{type}/{id}.json
func GenerateContentLocation(companySlug, docType, docID string) string {
	return fmt.Sprintf("https://%s.example/supply-chain/%s/%s.json", companySlug, docType, docID)
}

// ToSlug converts a company name to a URL-safe slug
// Example: "Pacific Silicon Foundry" -> "pacific-silicon-foundry"
func ToSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)
	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove common punctuation
	slug = strings.ReplaceAll(slug, ".", "")
	slug = strings.ReplaceAll(slug, ",", "")
	return slug
}
