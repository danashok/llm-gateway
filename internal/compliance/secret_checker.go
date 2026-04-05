package compliance

import (
	"context"
	"math"
	"regexp"
	"strings"

	"go.uber.org/zap"

	"github.com/ashokdan/llm-gateway/internal/models"
)

// secretPatterns defines regex patterns for detecting secrets
var secretPatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{"AWS Access Key", regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
	{"AWS Secret Key", regexp.MustCompile(`(?i)aws[_\-]?secret[_\-]?access[_\-]?key['":\s]*[=:]?\s*['"]?([A-Za-z0-9/+=]{40})['"]?`)},
	{"GitHub Token", regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`)},
	{"GitHub OAuth", regexp.MustCompile(`gho_[a-zA-Z0-9]{36}`)},
	{"GitHub App Token", regexp.MustCompile(`(ghu|ghs)_[a-zA-Z0-9]{36}`)},
	{"GitLab Token", regexp.MustCompile(`glpat-[a-zA-Z0-9\-]{20}`)},
	{"Slack Token", regexp.MustCompile(`xox[baprs]-[0-9]{10,13}-[0-9]{10,13}[a-zA-Z0-9-]*`)},
	{"Slack Webhook", regexp.MustCompile(`https://hooks\.slack\.com/services/T[a-zA-Z0-9_]+/B[a-zA-Z0-9_]+/[a-zA-Z0-9_]+`)},
	{"Google API Key", regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`)},
	{"Google OAuth", regexp.MustCompile(`[0-9]+-[0-9A-Za-z_]{32}\.apps\.googleusercontent\.com`)},
	{"Stripe API Key", regexp.MustCompile(`sk_live_[0-9a-zA-Z]{24}`)},
	{"Stripe Restricted Key", regexp.MustCompile(`rk_live_[0-9a-zA-Z]{24}`)},
	{"Heroku API Key", regexp.MustCompile(`(?i)heroku[_\-]?api[_\-]?key['":\s]*[=:]?\s*['"]?([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})['"]?`)},
	{"JWT Token", regexp.MustCompile(`eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*`)},
	{"Private Key", regexp.MustCompile(`-----BEGIN (RSA |DSA |EC |OPENSSH |PGP )?PRIVATE KEY( BLOCK)?-----`)},
	{"Generic Secret", regexp.MustCompile(`(?i)(api[_\-]?key|api[_\-]?secret|access[_\-]?token|auth[_\-]?token|credentials|passwd|password|secret)['":\s]*[=:]?\s*['"]([a-zA-Z0-9_\-]{16,64})['"]`)},
	{"OpenAI API Key", regexp.MustCompile(`sk-[a-zA-Z0-9]{48}`)},
	{"Anthropic API Key", regexp.MustCompile(`sk-ant-[a-zA-Z0-9\-]{95}`)},
	{"SendGrid API Key", regexp.MustCompile(`SG\.[a-zA-Z0-9_-]{22}\.[a-zA-Z0-9_-]{43}`)},
	{"Twilio API Key", regexp.MustCompile(`SK[0-9a-fA-F]{32}`)},
	{"NPM Token", regexp.MustCompile(`npm_[a-zA-Z0-9]{36}`)},
}

type secretChecker struct{}

// NewSecretChecker creates a new secret detection checker
func NewSecretChecker() IChecker {
	return &secretChecker{}
}

func (c *secretChecker) Name() string {
	return "secret_checker"
}

func (c *secretChecker) Check(ctx context.Context, logger *zap.Logger, prompt string) ([]Violation, error) {
	var violations []Violation

	// Check regex patterns
	for _, sp := range secretPatterns {
		matches := sp.pattern.FindAllStringIndex(prompt, -1)
		for _, match := range matches {
			matchedValue := prompt[match[0]:match[1]]
			violations = append(violations, Violation{
				CheckerName:  c.Name(),
				Severity:     models.SeverityCritical,
				Description:  "Detected potential " + sp.name,
				MatchedValue: matchedValue,
				StartIndex:   match[0],
				EndIndex:     match[1],
			})
		}
	}

	// Check for high-entropy strings (potential secrets)
	entropyViolations := c.checkHighEntropy(prompt)
	violations = append(violations, entropyViolations...)

	logger.Debug("secret check completed",
		zap.Int("violations_found", len(violations)),
	)

	return violations, nil
}

// checkHighEntropy looks for high-entropy strings that may be secrets
func (c *secretChecker) checkHighEntropy(text string) []Violation {
	var violations []Violation

	// Split text into words/tokens
	words := strings.Fields(text)
	for _, word := range words {
		// Skip short words or words with low entropy potential
		if len(word) < 20 || len(word) > 100 {
			continue
		}

		// Calculate Shannon entropy
		entropy := calculateEntropy(word)

		// High entropy threshold (typical for random secrets)
		if entropy > 4.5 {
			// Additional check: must have mix of character types
			hasUpper := strings.ContainsAny(word, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
			hasLower := strings.ContainsAny(word, "abcdefghijklmnopqrstuvwxyz")
			hasDigit := strings.ContainsAny(word, "0123456789")

			if (hasUpper && hasLower && hasDigit) || entropy > 5.0 {
				startIdx := strings.Index(text, word)
				violations = append(violations, Violation{
					CheckerName:  "secret_checker",
					Severity:     models.SeverityHigh,
					Description:  "Detected high-entropy string (potential secret)",
					MatchedValue: word,
					StartIndex:   startIdx,
					EndIndex:     startIdx + len(word),
				})
			}
		}
	}

	return violations
}

// calculateEntropy calculates the Shannon entropy of a string
func calculateEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	freq := make(map[rune]int)
	for _, c := range s {
		freq[c]++
	}

	var entropy float64
	length := float64(len(s))
	for _, count := range freq {
		p := float64(count) / length
		entropy -= p * math.Log2(p)
	}

	return entropy
}
