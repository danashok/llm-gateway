package compliance

import (
	"context"
	"regexp"

	"go.uber.org/zap"

	"github.com/ashokdan/llm-gateway/internal/models"
)

// piiPatterns defines regex patterns for detecting PII
var piiPatterns = []struct {
	name     string
	pattern  *regexp.Regexp
	severity models.ViolationSeverity
}{
	// Social Security Number (US)
	{"SSN", regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`), models.SeverityCritical},
	{"SSN (no dashes)", regexp.MustCompile(`\b\d{9}\b`), models.SeverityMedium}, // Lower severity due to false positives

	// Credit Card Numbers (Luhn-valid patterns)
	{"Credit Card (Visa)", regexp.MustCompile(`\b4\d{3}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`), models.SeverityCritical},
	{"Credit Card (Mastercard)", regexp.MustCompile(`\b5[1-5]\d{2}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`), models.SeverityCritical},
	{"Credit Card (Amex)", regexp.MustCompile(`\b3[47]\d{2}[\s-]?\d{6}[\s-]?\d{5}\b`), models.SeverityCritical},
	{"Credit Card (Discover)", regexp.MustCompile(`\b6(?:011|5\d{2})[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`), models.SeverityCritical},

	// Phone Numbers
	{"Phone (US)", regexp.MustCompile(`\b(?:\+1[-.\s]?)?\(?[2-9]\d{2}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b`), models.SeverityMedium},
	{"Phone (International)", regexp.MustCompile(`\b\+[1-9]\d{1,14}\b`), models.SeverityMedium},

	// Email Addresses
	{"Email", regexp.MustCompile(`\b[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b`), models.SeverityMedium},

	// IP Addresses
	{"IPv4 Address", regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`), models.SeverityLow},
	{"IPv6 Address", regexp.MustCompile(`\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b`), models.SeverityLow},

	// Date of Birth patterns
	{"Date of Birth", regexp.MustCompile(`\b(?i)(dob|date\s*of\s*birth|birth\s*date)[:\s]+\d{1,2}[/-]\d{1,2}[/-]\d{2,4}\b`), models.SeverityMedium},

	// Passport Numbers (various formats)
	{"Passport (US)", regexp.MustCompile(`\b[A-Z]\d{8}\b`), models.SeverityHigh},

	// Driver's License (US - varies by state, generic pattern)
	{"Driver's License", regexp.MustCompile(`(?i)(driver'?s?\s*license|dl)[:\s#]*[A-Z0-9]{5,15}\b`), models.SeverityHigh},

	// Bank Account Numbers (generic pattern)
	{"Bank Account", regexp.MustCompile(`(?i)(account\s*(?:number|#|no)?|acct)[:\s]*\d{8,17}\b`), models.SeverityCritical},

	// Routing Numbers (US)
	{"Routing Number", regexp.MustCompile(`(?i)(routing\s*(?:number|#|no)?|aba)[:\s]*\d{9}\b`), models.SeverityCritical},

	// Medicare/Medicaid Numbers
	{"Medicare Number", regexp.MustCompile(`\b[1-9][A-Z][A-Z0-9]\d-?[A-Z][A-Z0-9]\d-?[A-Z]{2}\d{2}\b`), models.SeverityCritical},

	// Tax ID / EIN
	{"Tax ID / EIN", regexp.MustCompile(`\b\d{2}-\d{7}\b`), models.SeverityHigh},
}

type piiChecker struct{}

// NewPIIChecker creates a new PII detection checker
func NewPIIChecker() IChecker {
	return &piiChecker{}
}

func (c *piiChecker) Name() string {
	return "pii_checker"
}

func (c *piiChecker) Check(ctx context.Context, logger *zap.Logger, prompt string) ([]Violation, error) {
	var violations []Violation

	for _, pp := range piiPatterns {
		matches := pp.pattern.FindAllStringIndex(prompt, -1)
		for _, match := range matches {
			matchedValue := prompt[match[0]:match[1]]

			// Skip common false positives
			if c.isFalsePositive(pp.name, matchedValue) {
				continue
			}

			violations = append(violations, Violation{
				CheckerName:  c.Name(),
				Severity:     pp.severity,
				Description:  "Detected potential " + pp.name,
				MatchedValue: matchedValue,
				StartIndex:   match[0],
				EndIndex:     match[1],
			})
		}
	}

	logger.Debug("PII check completed",
		zap.Int("violations_found", len(violations)),
	)

	return violations, nil
}

// isFalsePositive checks for common false positives
func (c *piiChecker) isFalsePositive(patternName, value string) bool {
	switch patternName {
	case "SSN (no dashes)":
		// Skip if it looks like a year or common number sequence
		if value == "000000000" || value == "123456789" {
			return true
		}
	case "IPv4 Address":
		// Skip localhost and common internal ranges
		if value == "127.0.0.1" || value == "0.0.0.0" {
			return true
		}
	case "Email":
		// Skip obvious example emails
		if value == "example@example.com" || value == "test@test.com" {
			return true
		}
	}
	return false
}
