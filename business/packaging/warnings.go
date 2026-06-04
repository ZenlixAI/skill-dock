package packaging

func warningDetails(codes []string) []WarningDetail {
	if len(codes) == 0 {
		return nil
	}

	details := make([]WarningDetail, 0, len(codes))
	for _, code := range codes {
		switch code {
		case "normalized_missing_package_json":
			details = append(details, WarningDetail{
				Code:            code,
				Severity:        "warning",
				Title:           "Generated missing package.json",
				Message:         "The imported skill did not provide a root package.json, so packaging generated one from the requested package_name, version, and metadata.",
				SuggestedAction: "Add and maintain a root package.json in the source skill if you want the package identity to live with the upstream source.",
			})
		case "missing_capability_content":
			details = append(details, WarningDetail{
				Code:            code,
				Severity:        "high-risk",
				Title:           "Missing capability content",
				Message:         "The imported skill did not contain a root SKILL.md or any recognized capability content such as commands/, agents/, hooks/, scripts/, docs/, plugin.json, or .mcp.json.",
				SuggestedAction: "Add a valid root SKILL.md or real capability content before validate/pack so the bundle contract is satisfied.",
			})
		case "normalized_rewrote_package_json_identity":
			details = append(details, WarningDetail{
				Code:            code,
				Severity:        "warning",
				Title:           "Rewrote package identity",
				Message:         "The imported skill already had a package.json, but its name and/or version did not match the requested internal target, so packaging rewrote those identity fields.",
				SuggestedAction: "Review whether the internal package_name/version mapping is what you want, and align the upstream package.json if this import should become the long-term source of truth.",
			})
		default:
			details = append(details, WarningDetail{
				Code:     code,
				Severity: "warning",
				Title:    code,
				Message:  code,
			})
		}
	}
	return details
}
