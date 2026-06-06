package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/google/uuid"

	projectsinfra "github.com/nebulacloud/nebula/internal/modules/projects/infrastructure"
	"github.com/nebulacloud/nebula/internal/platform/auth"
	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
)

// ListDomains lists custom hostnames for a service.
func (s *Service) ListDomains(ctx context.Context, actor auth.Principal, serviceID uuid.UUID) ([]projectsinfra.DomainRow, error) {
	if _, err := s.authorizeService(ctx, actor, serviceID, auth.RoleViewer); err != nil {
		return nil, err
	}
	return s.repo.ListDomainsByService(ctx, serviceID)
}

// CreateDomain registers a hostname (pending verification).
func (s *Service) CreateDomain(ctx context.Context, actor auth.Principal, serviceID uuid.UUID, hostname string) (projectsinfra.DomainRow, error) {
	if _, err := s.authorizeService(ctx, actor, serviceID, auth.RoleDeveloper); err != nil {
		return projectsinfra.DomainRow{}, err
	}
	host := normalizeHostname(hostname)
	if host == "" {
		return projectsinfra.DomainRow{}, platformerrors.Validation("hostname is required")
	}
	token, err := randomToken(16)
	if err != nil {
		return projectsinfra.DomainRow{}, platformerrors.Internal("token").WithCause(err)
	}
	d, err := s.repo.CreateDomain(ctx, serviceID, host, token)
	if err != nil {
		return projectsinfra.DomainRow{}, err
	}
	s.recordDomainAudit(ctx, actor, "domain.create", serviceID, map[string]any{"hostname": host})
	return d, nil
}

// DeleteDomain removes a custom domain.
func (s *Service) DeleteDomain(ctx context.Context, actor auth.Principal, domainID uuid.UUID) error {
	d, err := s.repo.GetDomain(ctx, domainID)
	if err != nil {
		return err
	}
	if _, err := s.authorizeService(ctx, actor, d.ServiceID, auth.RoleDeveloper); err != nil {
		return err
	}
	if err := s.repo.DeleteDomain(ctx, domainID); err != nil {
		return err
	}
	s.recordDomainAudit(ctx, actor, "domain.delete", d.ServiceID, map[string]any{"hostname": d.Hostname})
	return nil
}

// VerifyDomain marks domain verified (HTTP-01 via Traefik on next deploy).
func (s *Service) VerifyDomain(ctx context.Context, actor auth.Principal, domainID uuid.UUID) (projectsinfra.DomainRow, error) {
	d, err := s.repo.GetDomain(ctx, domainID)
	if err != nil {
		return projectsinfra.DomainRow{}, err
	}
	if _, err := s.authorizeService(ctx, actor, d.ServiceID, auth.RoleDeveloper); err != nil {
		return projectsinfra.DomainRow{}, err
	}
	out, err := s.repo.MarkDomainVerified(ctx, domainID)
	if err != nil {
		return projectsinfra.DomainRow{}, err
	}
	s.recordDomainAudit(ctx, actor, "domain.verify", d.ServiceID, map[string]any{"hostname": d.Hostname})
	return out, nil
}

func normalizeHostname(h string) string {
	h = strings.TrimSpace(strings.ToLower(h))
	h = strings.TrimPrefix(h, "https://")
	h = strings.TrimPrefix(h, "http://")
	h = strings.TrimSuffix(h, "/")
	return h
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Service) recordDomainAudit(ctx context.Context, actor auth.Principal, action string, serviceID uuid.UUID, meta map[string]any) {
	if s.recorder == nil {
		return
	}
	uid := actorFromPrincipal(actor)
	var actorPtr *uuid.UUID
	if uid != uuid.Nil {
		actorPtr = &uid
	}
	if meta == nil {
		meta = map[string]any{}
	}
	meta["service_id"] = serviceID.String()
	s.recorder.Record(ctx, action, actorPtr, meta)
}
