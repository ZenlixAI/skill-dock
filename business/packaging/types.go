package packaging

type Source struct {
	Type     string            `json:"type"`
	Location string            `json:"location,omitempty"`
	Ref      string            `json:"ref,omitempty"`
	Files    map[string]string `json:"files,omitempty"`
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

func (r PrepareRequest) GetTarget() Target {
	return r.Target
}

type PrepareResponse struct {
	WorkspacePath string   `json:"workspacePath"`
	PackageName   string   `json:"packageName"`
	Version       string   `json:"version"`
	Warnings      []string `json:"warnings"`
	Errors        []string `json:"errors"`
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
	OK             bool     `json:"ok"`
	WorkspacePath  string   `json:"workspacePath"`
	PackageName    string   `json:"packageName"`
	Version        string   `json:"version"`
	ManifestDigest string   `json:"manifestDigest,omitempty"`
	Components     []string `json:"components,omitempty"`
	Warnings       []string `json:"warnings"`
	Errors         []string `json:"errors"`
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
	WorkspacePath  string   `json:"workspacePath"`
	BundlePath     string   `json:"bundlePath"`
	BundleBase64   string   `json:"bundleBase64,omitempty"`
	PackageName    string   `json:"packageName"`
	Version        string   `json:"version"`
	ManifestDigest string   `json:"manifestDigest"`
	Components     []string `json:"components"`
	Warnings       []string `json:"warnings"`
	Errors         []string `json:"errors"`
}
