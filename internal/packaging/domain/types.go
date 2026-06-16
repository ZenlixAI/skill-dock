package domain

import catalog "skill_registry/internal/catalog/domain"

type Source struct {
	Type          string            `json:"type"`
	Location      string            `json:"location,omitempty"`
	Ref           string            `json:"ref,omitempty"`
	Files         map[string]string `json:"files,omitempty"`
	ArchiveBase64 string            `json:"archiveBase64,omitempty"`
}

type Target struct {
	PackageName string `json:"packageName"`
	Version     string `json:"version"`
}

type Metadata struct {
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	SourceType  string `json:"sourceType,omitempty"`
}

type Options struct {
	PublishRegistry string `json:"publishRegistry,omitempty"`
	Strict          bool   `json:"strict,omitempty"`
}

type WarningDetail struct {
	Code            string `json:"code"`
	Severity        string `json:"severity"`
	Title           string `json:"title"`
	Message         string `json:"message"`
	SuggestedAction string `json:"suggestedAction,omitempty"`
}

type ImportReport struct {
	SourceType                     string   `json:"sourceType,omitempty"`
	StrictMode                     bool     `json:"strictMode"`
	Normalized                     bool     `json:"normalized"`
	GeneratedPackageJSON           bool     `json:"generatedPackageJson"`
	GeneratedCapabilityPlaceholder bool     `json:"generatedCapabilityPlaceholder"`
	RewrotePackageIdentity         bool     `json:"rewrotePackageIdentity"`
	GeneratedPaths                 []string `json:"generatedPaths,omitempty"`
	WarningCodes                   []string `json:"warningCodes,omitempty"`
}

type ReviewSummary struct {
	ManifestDigest string   `json:"manifestDigest,omitempty"`
	Components     []string `json:"components,omitempty"`
}

type ImportReviewRequest struct {
	IdempotencyKey      string   `json:"idempotencyKey,omitempty"`
	Source              Source   `json:"source"`
	Target              Target   `json:"target"`
	Metadata            Metadata `json:"metadata"`
	Options             Options  `json:"options"`
	IncludeBundleBase64 bool     `json:"includeBundleBase64,omitempty"`
}

func (r ImportReviewRequest) GetIdempotencyKey() string {
	return r.IdempotencyKey
}

func (r ImportReviewRequest) GetSource() Source {
	return r.Source
}

func (r ImportReviewRequest) GetTarget() Target {
	return r.Target
}

type ImportReviewResponse struct {
	OK             bool            `json:"ok"`
	PackageName    string          `json:"packageName"`
	Version        string          `json:"version"`
	ManifestDigest string          `json:"manifestDigest,omitempty"`
	Components     []string        `json:"components,omitempty"`
	BundleBase64   string          `json:"bundleBase64,omitempty"`
	Warnings       []string        `json:"warnings"`
	WarningDetails []WarningDetail `json:"warningDetails,omitempty"`
	ImportReport   *ImportReport   `json:"importReport,omitempty"`
	Errors         []string        `json:"errors"`
}

type PrepareRequest struct {
	IdempotencyKey string   `json:"idempotencyKey,omitempty"`
	Source         Source   `json:"source"`
	Target         Target   `json:"target"`
	Metadata       Metadata `json:"metadata"`
	Options        Options  `json:"options"`
}

func (r PrepareRequest) GetIdempotencyKey() string {
	return r.IdempotencyKey
}

func (r PrepareRequest) GetSource() Source {
	return r.Source
}

func (r PrepareRequest) GetTarget() Target {
	return r.Target
}

type PrepareResponse struct {
	WorkspacePath  string          `json:"workspacePath"`
	PackageName    string          `json:"packageName"`
	Version        string          `json:"version"`
	Warnings       []string        `json:"warnings"`
	WarningDetails []WarningDetail `json:"warningDetails,omitempty"`
	ImportReport   *ImportReport   `json:"importReport,omitempty"`
	Errors         []string        `json:"errors"`
}

type ValidateRequest struct {
	IdempotencyKey string `json:"idempotencyKey,omitempty"`
	WorkspacePath  string `json:"workspacePath"`
	Target         Target `json:"target"`
}

func (r ValidateRequest) GetIdempotencyKey() string {
	return r.IdempotencyKey
}

func (r ValidateRequest) GetTarget() Target {
	return r.Target
}

type ValidateResponse struct {
	OK             bool            `json:"ok"`
	WorkspacePath  string          `json:"workspacePath"`
	PackageName    string          `json:"packageName"`
	Version        string          `json:"version"`
	ManifestDigest string          `json:"manifestDigest,omitempty"`
	Components     []string        `json:"components,omitempty"`
	Warnings       []string        `json:"warnings"`
	WarningDetails []WarningDetail `json:"warningDetails,omitempty"`
	ImportReport   *ImportReport   `json:"importReport,omitempty"`
	Errors         []string        `json:"errors"`
}

type PackRequest struct {
	IdempotencyKey string `json:"idempotencyKey,omitempty"`
	WorkspacePath  string `json:"workspacePath"`
	Target         Target `json:"target"`
	IncludeBundle  bool   `json:"includeBundle,omitempty"`
}

func (r PackRequest) GetIdempotencyKey() string {
	return r.IdempotencyKey
}

func (r PackRequest) GetTarget() Target {
	return r.Target
}

type PackResponse struct {
	WorkspacePath  string          `json:"workspacePath"`
	BundlePath     string          `json:"bundlePath"`
	BundleBase64   string          `json:"bundleBase64,omitempty"`
	PackageName    string          `json:"packageName"`
	Version        string          `json:"version"`
	ManifestDigest string          `json:"manifestDigest"`
	Components     []string        `json:"components"`
	Warnings       []string        `json:"warnings"`
	WarningDetails []WarningDetail `json:"warningDetails,omitempty"`
	ImportReport   *ImportReport   `json:"importReport,omitempty"`
	Errors         []string        `json:"errors"`
}

type PublishRequest struct {
	IdempotencyKey  string        `json:"idempotencyKey,omitempty"`
	Source          Source        `json:"source"`
	Target          Target        `json:"target"`
	Metadata        Metadata      `json:"metadata"`
	Options         Options       `json:"options"`
	Review          ReviewSummary `json:"review,omitempty"`
}

func (r PublishRequest) GetTarget() Target {
	return r.Target
}

type PublishResponse struct {
	WorkspacePath       string                `json:"workspacePath"`
	PackageName         string                `json:"packageName"`
	Version             string                `json:"version"`
	ManifestDigest      string                `json:"manifestDigest"`
	Components          []string              `json:"components"`
	PublishedToRegistry bool                  `json:"publishedToRegistry"`
	Created             bool                  `json:"created"`
	IdempotentReplay    bool                  `json:"idempotentReplay"`
	VersionStatus       catalog.VersionStatus `json:"versionStatus"`
	InstallPolicy       catalog.InstallPolicy `json:"installPolicy"`
	Warnings            []string              `json:"warnings"`
	WarningDetails      []WarningDetail       `json:"warningDetails,omitempty"`
	ImportReport        *ImportReport         `json:"importReport,omitempty"`
	Errors              []string              `json:"errors"`
}

type CleanupRequest struct {
	WorkspacePath string `json:"workspacePath"`
}

type CleanupResponse struct {
	WorkspacePath string `json:"workspacePath"`
	Removed       bool   `json:"removed"`
}
