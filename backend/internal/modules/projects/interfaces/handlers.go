package interfaces

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	projectsapp "github.com/nebulacloud/nebula/internal/modules/projects/application"
	projectsinfra "github.com/nebulacloud/nebula/internal/modules/projects/infrastructure"
	"github.com/nebulacloud/nebula/internal/platform/auth"
	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
	"github.com/nebulacloud/nebula/internal/platform/httpx"
	"github.com/nebulacloud/nebula/internal/platform/secrets"
)

// Handler exposes Phase 2+ workspace HTTP endpoints.
type Handler struct {
	svc    *projectsapp.Service
	sealer secrets.Sealer
	rtBase string // NEBULA_BASE_DOMAIN
}

// NewHandler builds a Handler wired for JSON + secrets decryption previews.
func NewHandler(svc *projectsapp.Service, sealer secrets.Sealer, runtimeBaseDomain string) *Handler {
	return &Handler{svc: svc, sealer: sealer, rtBase: strings.TrimSpace(runtimeBaseDomain)}
}

// Mount installs authenticated workspace routes under r (mount at `/` inside authed `/api/v1`).
func (h *Handler) Mount(r chi.Router) {
	r.Get("/organizations", h.listOrganizations)
	r.Post("/organizations", h.createOrganization)

	r.Route("/organizations/{orgID}", func(rr chi.Router) {
		rr.Get("/projects", h.listProjects)
		rr.Post("/projects", h.createProject)
	})

	r.Get("/projects/{projectID}", h.getProject)
	r.Patch("/projects/{projectID}", h.patchProject)

	r.Route("/projects/{projectID}", func(rr chi.Router) {
		rr.Get("/services", h.listServices)
		rr.Post("/services", h.createService)
		rr.Get("/deployments", h.listDeploymentsByProject)
	})

	r.Get("/services/{serviceID}", h.getService)
	r.Patch("/services/{serviceID}", h.patchService)

	r.Route("/services/{serviceID}", func(rr chi.Router) {
		rr.Get("/env-vars", h.listEnv)
		rr.Post("/env-vars", h.postEnvVar)
		rr.Delete("/env-vars/{envKey}", h.deleteEnvVar)
		rr.Get("/deployments", h.listDeployments)
		rr.Post("/deployments", h.createDeployment)
		rr.Get("/logs", h.serviceLogs)
		rr.Get("/metrics", h.serviceMetrics)
	})

	r.Get("/deployments/{deploymentID}", h.getDeployment)

	r.Post("/integrations/github/installation", h.StubGithubInstallationLink)
}

func principal(r *http.Request) (auth.Principal, bool) {
	return auth.PrincipalFromContext(r.Context())
}

func parseUUID(seg string, w http.ResponseWriter) (uuid.UUID, bool) {
	id, err := uuid.Parse(seg)
	if err != nil {
		httpx.Error(w, platformerrors.Validation("invalid id"))
		return uuid.Nil, false
	}
	return id, true
}

func qInt(r *http.Request, k string, def int) int {
	s := strings.TrimSpace(r.URL.Query().Get(k))
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func qDur(r *http.Request, k string, def time.Duration) time.Duration {
	s := strings.TrimSpace(r.URL.Query().Get(k))
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}

// --- Organizations ---

func (h *Handler) listOrganizations(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	rows, err := h.svc.ListOrganizations(r.Context(), pr)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	out := make([]organizationDTO, 0, len(rows))
	for _, o := range rows {
		out = append(out, toOrgDTO(o))
	}
	httpx.OK(w, out)
}

type organizationDTO struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Plan      string `json:"plan"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toOrgDTO(o projectsinfra.OrganizationRow) organizationDTO {
	return organizationDTO{
		ID:        o.ID.String(),
		Slug:      o.Slug,
		Name:      o.Name,
		Plan:      o.Plan,
		CreatedAt: o.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: o.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

type createOrgBody struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

func (h *Handler) createOrganization(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	var body createOrgBody
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Error(w, err)
		return
	}
	row, err := h.svc.CreateOrganization(r.Context(), pr, body.Slug, body.Name)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, toOrgDTO(row))
}

// --- Projects ---

type projectDTO struct {
	ID               string  `json:"id"`
	OrganizationID   string  `json:"organization_id"`
	Slug             string  `json:"slug"`
	Name             string  `json:"name"`
	Description      *string `json:"description,omitempty"`
	RepoURL          *string `json:"repo_url,omitempty"`
	DefaultBranch    string  `json:"default_branch"`
	ServicesCount    int     `json:"services_count"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

func toProjectDTO(p projectsinfra.ProjectRow, svcCount int) projectDTO {
	return projectDTO{
		ID:             p.ID.String(),
		OrganizationID: p.OrganizationID.String(),
		Slug:           p.Slug,
		Name:           p.Name,
		Description:    p.Description,
		RepoURL:        p.RepoURL,
		DefaultBranch:  p.DefaultBranch,
		ServicesCount:    svcCount,
		CreatedAt:      p.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:      p.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	orgID, ok2 := parseUUID(chi.URLParam(r, "orgID"), w)
	if !ok2 {
		return
	}
	projs, err := h.svc.ListProjects(r.Context(), pr, orgID)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	out := make([]projectDTO, 0, len(projs))
	for _, p := range projs {
		cnt, err := h.svc.CountServices(r.Context(), pr, p.ID)
		if err != nil {
			httpx.Error(w, err)
			return
		}
		out = append(out, toProjectDTO(p, cnt))
	}
	httpx.OK(w, out)
}

type createProjectBody struct {
	Slug           string   `json:"slug"`
	Name           string   `json:"name"`
	Description    *string  `json:"description,omitempty"`
	RepoURL        *string  `json:"repo_url,omitempty"`
	DefaultBranch string   `json:"default_branch,omitempty"`
}

func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	orgID, ok2 := parseUUID(chi.URLParam(r, "orgID"), w)
	if !ok2 {
		return
	}
	var body createProjectBody
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Error(w, err)
		return
	}
	row, err := h.svc.CreateProject(r.Context(), pr, orgID, body.Slug, body.Name, body.Description, body.RepoURL, body.DefaultBranch)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, toProjectDTO(row, 0))
}

type patchProjectBody struct {
	Name          *string `json:"name,omitempty"`
	Description   *string `json:"description,omitempty"`
	RepoURL       *string `json:"repo_url,omitempty"`
	DefaultBranch *string `json:"default_branch,omitempty"`
}

func (h *Handler) patchProject(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	pid, ok2 := parseUUID(chi.URLParam(r, "projectID"), w)
	if !ok2 {
		return
	}
	var body patchProjectBody
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Error(w, err)
		return
	}
	row, err := h.svc.UpdateProject(r.Context(), pr, pid, body.Name, body.Description, body.RepoURL, body.DefaultBranch)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	cnt, err := h.svc.CountServices(r.Context(), pr, pid)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, toProjectDTO(row, cnt))
}

func (h *Handler) getProject(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	pid, ok2 := parseUUID(chi.URLParam(r, "projectID"), w)
	if !ok2 {
		return
	}
	row, err := h.svc.GetProject(r.Context(), pr, pid)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	cnt, err := h.svc.CountServices(r.Context(), pr, pid)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, toProjectDTO(row, cnt))
}

// --- Services ---

type serviceDTO struct {
	ID            string `json:"id"`
	ProjectID     string `json:"project_id"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	CurrentImage  *string `json:"current_image,omitempty"`
	URL           *string `json:"url,omitempty"`
	Replicas      int    `json:"replicas"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

func toServiceDTO(s projectsinfra.ServiceRow, baseDomain string, projSlug string) serviceDTO {
	dto := serviceDTO{
		ID:           s.ID.String(),
		ProjectID:    s.ProjectID.String(),
		Slug:         s.Slug,
		Name:         s.Name,
		Type:         s.Type,
		Status:       s.Status,
		CurrentImage: s.CurrentImage,
		Replicas:     projectsapp.Replicas(s.RuntimeConfig),
		CreatedAt:    s.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:    s.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if s.Type == "web" && baseDomain != "" && projSlug != "" {
		u := fmtServiceURL(baseDomain, s.Slug, projSlug)
		dto.URL = &u
	}
	return dto
}

func fmtServiceURL(baseDomain, svcSlug, projSlug string) string {
	host := sanitizeHost(svcSlug) + "." + sanitizeHost(projSlug) + "." + strings.Trim(strings.TrimPrefix(baseDomain, "."), "")
	return "http://" + host
}

func sanitizeHost(s string) string {
	return strings.Trim(strings.ToLower(s), "")
}

func (h *Handler) listServices(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	pid, ok2 := parseUUID(chi.URLParam(r, "projectID"), w)
	if !ok2 {
		return
	}
	rows, err := h.svc.ListServices(r.Context(), pr, pid)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	proj, err := h.svc.GetProject(r.Context(), pr, pid)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	out := make([]serviceDTO, 0, len(rows))
	for _, svc := range rows {
		out = append(out, toServiceDTO(svc, h.rtBase, proj.Slug))
	}
	httpx.OK(w, out)
}

type createServiceBody struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

func (h *Handler) createService(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	pid, ok2 := parseUUID(chi.URLParam(r, "projectID"), w)
	if !ok2 {
		return
	}
	var body createServiceBody
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Error(w, err)
		return
	}
	row, err := h.svc.CreateService(r.Context(), pr, pid, body.Slug, body.Name, body.Type)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	proj, err := h.svc.GetProject(r.Context(), pr, pid)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, toServiceDTO(row, h.rtBase, proj.Slug))
}

func (h *Handler) getService(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	sid, ok2 := parseUUID(chi.URLParam(r, "serviceID"), w)
	if !ok2 {
		return
	}
	row, err := h.svc.GetService(r.Context(), pr, sid)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	proj, err := h.svc.GetProject(r.Context(), pr, row.ProjectID)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, toServiceDTO(row, h.rtBase, proj.Slug))
}

type patchServiceBody struct {
	Name          *string          `json:"name,omitempty"`
	BuildConfig   *json.RawMessage `json:"build_config,omitempty"`
	RuntimeConfig *json.RawMessage `json:"runtime_config,omitempty"`
}

func (h *Handler) patchService(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	sid, ok2 := parseUUID(chi.URLParam(r, "serviceID"), w)
	if !ok2 {
		return
	}
	var body patchServiceBody
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Error(w, err)
		return
	}
	var bp, rtp json.RawMessage
	if body.BuildConfig != nil {
		bp = *body.BuildConfig
	}
	if body.RuntimeConfig != nil {
		rtp = *body.RuntimeConfig
	}
	row, err := h.svc.UpdateService(r.Context(), pr, sid, body.Name, bp, rtp)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	proj, err := h.svc.GetProject(r.Context(), pr, row.ProjectID)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, toServiceDTO(row, h.rtBase, proj.Slug))
}

// --- env ---

type envVarDTO struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	IsSecret  bool   `json:"is_secret"`
	Preview   string `json:"preview,omitempty"`
	UpdatedAt string `json:"updated_at"`
}

func (h *Handler) sealPreview(b []byte) string {
	pt, err := h.sealer.Open(b)
	if err != nil || len(pt) == 0 {
		return ""
	}
	s := string(pt)
	if len(s) <= 6 {
		return s
	}
	return "…" + s[len(s)-4:]
}

func toEnvDTO(e projectsinfra.EnvVarRow, preview string, updated projectsinfra.EnvVarRow) envVarDTO {
	return envVarDTO{
		ID:        e.ID.String(),
		Key:       e.Key,
		IsSecret:  e.IsSecret,
		Preview:   preview,
		UpdatedAt: e.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (h *Handler) listEnv(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	sid, ok2 := parseUUID(chi.URLParam(r, "serviceID"), w)
	if !ok2 {
		return
	}
	rows, err := h.svc.ListEnv(r.Context(), pr, sid)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	out := make([]envVarDTO, 0, len(rows))
	for _, e := range rows {
		p := h.sealPreview(e.ValueEnc)
		out = append(out, toEnvDTO(e, p, e))
	}
	httpx.OK(w, out)
}

type upsertEnvBody struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret bool   `json:"is_secret,omitempty"`
}

func (h *Handler) postEnvVar(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	sid, ok2 := parseUUID(chi.URLParam(r, "serviceID"), w)
	if !ok2 {
		return
	}
	var body upsertEnvBody
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Error(w, err)
		return
	}
	row, err := h.svc.UpsertEnvVar(r.Context(), pr, sid, body.Key, body.Value, body.IsSecret)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	p := h.sealPreview(row.ValueEnc)
	httpx.Created(w, toEnvDTO(row, p, row))
}

func (h *Handler) deleteEnvVar(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	sid, ok2 := parseUUID(chi.URLParam(r, "serviceID"), w)
	if !ok2 {
		return
	}
	key := chi.URLParam(r, "envKey")
	if key == "" {
		httpx.Error(w, platformerrors.Validation("missing env key"))
		return
	}
	if err := h.svc.DeleteEnvVar(r.Context(), pr, sid, key); err != nil {
		httpx.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- deployments ---

type deploymentDTO struct {
	ID            string               `json:"id"`
	ServiceID     string               `json:"service_id"`
	ServiceName   string               `json:"service_name"`
	ProjectID     string               `json:"project_id"`
	ProjectName   string               `json:"project_name"`
	Trigger       string               `json:"trigger"`
	Status        string               `json:"status"`
	CommitSHA     *string              `json:"commit_sha,omitempty"`
	CommitMessage *string              `json:"commit_message,omitempty"`
	Ref           *string              `json:"ref,omitempty"`
	ImageRef      *string              `json:"image_ref,omitempty"`
	DurationMs    int64                `json:"duration_ms,omitempty"`
	CreatedAt     string               `json:"created_at"`
	StartedAt     *string              `json:"started_at,omitempty"`
	FinishedAt    *string              `json:"finished_at,omitempty"`
	TriggeredBy   *deploymentActorDTO  `json:"triggered_by,omitempty"`
}

type deploymentActorDTO struct {
	ID    string `json:"id"`
	Email string `json:"email,omitempty"`
}

func timePtrRFC(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339Nano)
	return &s
}

func (h *Handler) listDeploymentsByProject(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	pid, ok2 := parseUUID(chi.URLParam(r, "projectID"), w)
	if !ok2 {
		return
	}
	items, err := h.svc.ListDeployments(r.Context(), pr, nil, &pid, qInt(r, "limit", 50))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	out := make([]deploymentDTO, 0, len(items))
	for _, it := range items {
		out = append(out, toDeployDTO(it))
	}
	httpx.OK(w, out)
}

func toDeployDTO(j projectsinfra.DeploymentJoined) deploymentDTO {
	d := deploymentDTO{
		ID:           j.ID.String(),
		ServiceID:    j.ServiceID.String(),
		ServiceName:  j.ServiceName,
		ProjectID:    j.ProjectID.String(),
		ProjectName:  j.ProjectName,
		Trigger:      j.Trigger,
		Status:       j.Status,
		CommitSHA:    j.CommitSHA,
		DurationMs:   j.DurationMs,
		ImageRef:     j.ImageRef,
		CreatedAt:    j.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if j.CommitMsg != nil {
		m := *j.CommitMsg
		d.CommitMessage = &m
	}
	d.Ref = j.Ref
	d.StartedAt = timePtrRFC(j.StartedAt)
	d.FinishedAt = timePtrRFC(j.FinishedAt)
	if j.TriggeredBy != nil && j.TriggerEmail != nil {
		d.TriggeredBy = &deploymentActorDTO{ID: j.TriggeredBy.String(), Email: *j.TriggerEmail}
	}
	return d
}

func (h *Handler) listDeployments(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	sidRaw := chi.URLParam(r, "serviceID")
	sid, err := uuid.Parse(sidRaw)
	var ptr *uuid.UUID
	if err == nil {
		ptr = &sid
	}
	lim := qInt(r, "limit", 50)
	items, err := h.svc.ListDeployments(r.Context(), pr, ptr, nil, lim)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	out := make([]deploymentDTO, 0, len(items))
	for _, it := range items {
		out = append(out, toDeployDTO(it))
	}
	httpx.OK(w, out)
}

func (h *Handler) createDeployment(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	sid, ok2 := parseUUID(chi.URLParam(r, "serviceID"), w)
	if !ok2 {
		return
	}
	depID, err := h.svc.StartDeployment(r.Context(), pr, sid)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, map[string]string{"id": depID.String(), "status": "queued"})
}

func (h *Handler) getDeployment(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	id, ok2 := parseUUID(chi.URLParam(r, "deploymentID"), w)
	if !ok2 {
		return
	}
	dj, err := h.svc.GetDeployment(r.Context(), pr, id)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, toDeployDTO(dj))
}

func (h *Handler) serviceLogs(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	sid, ok2 := parseUUID(chi.URLParam(r, "serviceID"), w)
	if !ok2 {
		return
	}
	win := qDur(r, "window", 15*time.Minute)
	lines, err := h.svc.QueryLogs(r.Context(), pr, sid, win, qInt(r, "limit", 200))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, lines)
}

func (h *Handler) serviceMetrics(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	sid, ok2 := parseUUID(chi.URLParam(r, "serviceID"), w)
	if !ok2 {
		return
	}
	win := qDur(r, "window", time.Hour)
	series, err := h.svc.QueryMetrics(r.Context(), pr, sid, win)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, series)
}

