package interfaces

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	projectsinfra "github.com/nebulacloud/nebula/internal/modules/projects/infrastructure"
	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
	"github.com/nebulacloud/nebula/internal/platform/httpx"
)

type domainDTO struct {
	ID                string  `json:"id"`
	ServiceID         string  `json:"service_id"`
	Hostname          string  `json:"hostname"`
	IsPrimary         bool    `json:"is_primary"`
	SSLStatus         string  `json:"ssl_status"`
	VerificationToken string  `json:"verification_token,omitempty"`
	VerifiedAt        *string `json:"verified_at,omitempty"`
	CNAMEHint         string  `json:"cname_hint,omitempty"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

func toDomainDTO(d projectsinfra.DomainRow, baseDomain string) domainDTO {
	out := domainDTO{
		ID:                d.ID.String(),
		ServiceID:         d.ServiceID.String(),
		Hostname:          d.Hostname,
		IsPrimary:         d.IsPrimary,
		SSLStatus:         d.SSLStatus,
		VerificationToken: d.VerificationToken,
		CreatedAt:         d.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:         d.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if d.VerifiedAt != nil {
		s := d.VerifiedAt.UTC().Format(time.RFC3339Nano)
		out.VerifiedAt = &s
	}
	if baseDomain != "" {
		out.CNAMEHint = "CNAME " + d.Hostname + " → your-service." + baseDomain
	}
	return out
}

func (h *Handler) listDomains(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	sid, ok2 := parseUUID(chi.URLParam(r, "serviceID"), w)
	if !ok2 {
		return
	}
	rows, err := h.svc.ListDomains(r.Context(), pr, sid)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	out := make([]domainDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainDTO(row, h.rtBase))
	}
	httpx.OK(w, out)
}

type createDomainBody struct {
	Hostname string `json:"hostname"`
}

func (h *Handler) createDomain(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	sid, ok2 := parseUUID(chi.URLParam(r, "serviceID"), w)
	if !ok2 {
		return
	}
	var body createDomainBody
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Error(w, err)
		return
	}
	row, err := h.svc.CreateDomain(r.Context(), pr, sid, body.Hostname)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Created(w, toDomainDTO(row, h.rtBase))
}

func (h *Handler) deleteDomain(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	did, ok2 := parseUUID(chi.URLParam(r, "domainID"), w)
	if !ok2 {
		return
	}
	if err := h.svc.DeleteDomain(r.Context(), pr, did); err != nil {
		httpx.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) verifyDomain(w http.ResponseWriter, r *http.Request) {
	pr, ok := principal(r)
	if !ok {
		httpx.Error(w, platformerrors.Unauthorized("not authenticated"))
		return
	}
	did, ok2 := parseUUID(chi.URLParam(r, "domainID"), w)
	if !ok2 {
		return
	}
	row, err := h.svc.VerifyDomain(r.Context(), pr, did)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.OK(w, toDomainDTO(row, h.rtBase))
}
