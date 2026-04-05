package compliance

import (
	"context"
	"regexp"
	"strings"

	"go.uber.org/zap"

	"github.com/ashokdan/llm-gateway/internal/models"
)

// sensitiveProjectPatterns defines patterns for detecting sensitive aerospace project references
var sensitiveProjectPatterns = []struct {
	name     string
	patterns []*regexp.Regexp
	severity models.ViolationSeverity
}{
	{
		name: "Classified Project Reference",
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(classified|top\s*secret|secret|confidential)\s+(project|program|mission|operation)\b`),
			regexp.MustCompile(`(?i)\b(sap|sci|noforn|orcon|rel\s*to)\b`), // Security classifications
		},
		severity: models.SeverityCritical,
	},
	{
		name: "Export Controlled Content",
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(itar|ear|export\s*control|eccn|usml)\b`),
			regexp.MustCompile(`(?i)\b(controlled\s*unclassified|cui|fouo)\b`),
		},
		severity: models.SeverityCritical,
	},
	{
		name: "Defense Program Reference",
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(darpa|dod|defense\s*advanced|pentagon)\s+(project|program|contract)\b`),
			regexp.MustCompile(`(?i)\b(contract\s*number|cage\s*code|duns)\s*[:\s]*[A-Z0-9\-]+\b`),
		},
		severity: models.SeverityHigh,
	},
	{
		name: "Sensitive Technical Data",
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(propulsion\s*system|guidance\s*system|targeting\s*system|radar\s*signature)\b`),
			regexp.MustCompile(`(?i)\b(stealth|radar\s*cross\s*section|rcs|low\s*observable)\s+(technology|design|specification)\b`),
			regexp.MustCompile(`(?i)\b(encryption\s*key|cipher|cryptographic)\s+(algorithm|specification|implementation)\b`),
		},
		severity: models.SeverityHigh,
	},
	{
		name: "Internal Project Code",
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\bproject\s*[:\s]*(eagle|falcon|phoenix|titan|atlas|orion|apollo)\s*\d*\b`),
			regexp.MustCompile(`(?i)\b(internal\s*only|company\s*confidential|proprietary)\b`),
		},
		severity: models.SeverityMedium,
	},
	{
		name: "Facility Reference",
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(scif|skif|sensitive\s*compartmented\s*information\s*facility)\b`),
			regexp.MustCompile(`(?i)\b(clean\s*room|secure\s*area|restricted\s*zone)\s*[A-Z0-9\-]+\b`),
		},
		severity: models.SeverityMedium,
	},
}

// blockedKeywords that should never be sent to external LLMs
var blockedKeywords = []string{
	"nuclear",
	"weapons grade",
	"warhead",
	"missile defense",
	"icbm",
	"hypersonic",
	"anti-satellite",
	"directed energy",
	"emp",
	"electromagnetic pulse",
}

type projectChecker struct{}

// NewProjectChecker creates a new sensitive project detection checker
func NewProjectChecker() IChecker {
	return &projectChecker{}
}

func (c *projectChecker) Name() string {
	return "project_checker"
}

func (c *projectChecker) Check(ctx context.Context, logger *zap.Logger, prompt string) ([]Violation, error) {
	var violations []Violation

	// Check regex patterns for sensitive project references
	for _, sp := range sensitiveProjectPatterns {
		for _, pattern := range sp.patterns {
			matches := pattern.FindAllStringIndex(prompt, -1)
			for _, match := range matches {
				matchedValue := prompt[match[0]:match[1]]
				violations = append(violations, Violation{
					CheckerName:  c.Name(),
					Severity:     sp.severity,
					Description:  "Detected " + sp.name,
					MatchedValue: matchedValue,
					StartIndex:   match[0],
					EndIndex:     match[1],
				})
			}
		}
	}

	// Check for blocked keywords
	lowerPrompt := strings.ToLower(prompt)
	for _, keyword := range blockedKeywords {
		if idx := strings.Index(lowerPrompt, keyword); idx != -1 {
			violations = append(violations, Violation{
				CheckerName:  c.Name(),
				Severity:     models.SeverityCritical,
				Description:  "Detected blocked keyword related to sensitive defense content",
				MatchedValue: keyword,
				StartIndex:   idx,
				EndIndex:     idx + len(keyword),
			})
		}
	}

	logger.Debug("project check completed",
		zap.Int("violations_found", len(violations)),
	)

	return violations, nil
}
