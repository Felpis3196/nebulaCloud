// Package jobs defines shared job payloads for asynq tasks consumed by the
// API, build worker, and runtime agent.
package jobs

// Asynq task type names (stable contract).
const (
	TaskBuild  = "build:run"
	TaskDeploy = "deploy:run"
)

// BuildRunPayload is enqueued when a deployment should be built.
type BuildRunPayload struct {
	BuildID        string `json:"build_id"`
	DeploymentID   string `json:"deployment_id"`
	ServiceID      string `json:"service_id"`
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	RepoURL        string `json:"repo_url"`
	Ref            string `json:"ref"`
	RegistryURL    string `json:"registry_url"`
	RegistryInsecure bool `json:"registry_insecure"`
	ImageRef       string `json:"image_ref"`
	DockerfilePath string `json:"dockerfile_path,omitempty"`
}

// DeployRunPayload instructs the runtime agent to run a built image.
type DeployRunPayload struct {
	DeploymentID   string `json:"deployment_id"`
	ServiceID      string `json:"service_id"`
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	OrgSlug        string `json:"org_slug"`
	ProjectSlug    string `json:"project_slug"`
	ServiceSlug    string `json:"service_slug"`
	ImageRef       string `json:"image_ref"`
	ListenPort     int    `json:"listen_port"`
	DetectedStack  string `json:"detected_stack,omitempty"`
	BaseDomain     string `json:"base_domain"`
}
