// Package validator centralises lightweight input validation helpers shared
// across modules. The platform deliberately avoids pulling in a heavyweight
// reflection-based validator at this stage so the platform layer stays
// dependency-light; modules can add tag-based validators if/when a clear
// need emerges.
package validator

import (
	"net/mail"
	"regexp"
	"strings"

	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]$`)

// Builder accumulates field-level validation errors and returns a single
// platform error when Done is called.
//
//	v := validator.New()
//	v.Required("email", req.Email)
//	v.Email("email", req.Email)
//	v.MaxLen("name", req.Name, 64)
//	if err := v.Done(); err != nil { return err }
type Builder struct {
	err *platformerrors.Error
}

// New returns an empty validator builder.
func New() *Builder { return &Builder{err: platformerrors.Validation("invalid input")} }

// Required reports an error if value is empty (after trimming whitespace).
func (b *Builder) Required(field, value string) *Builder {
	if strings.TrimSpace(value) == "" {
		b.err.WithDetail(field, "is required")
	}
	return b
}

// MinLen enforces a minimum length on the trimmed value.
func (b *Builder) MinLen(field, value string, min int) *Builder {
	if len(strings.TrimSpace(value)) < min {
		b.err.WithDetail(field, "is too short")
	}
	return b
}

// MaxLen enforces a maximum length on the trimmed value.
func (b *Builder) MaxLen(field, value string, max int) *Builder {
	if len(value) > max {
		b.err.WithDetail(field, "is too long")
	}
	return b
}

// Email enforces RFC 5322 email syntax.
func (b *Builder) Email(field, value string) *Builder {
	if _, err := mail.ParseAddress(value); err != nil {
		b.err.WithDetail(field, "is not a valid email")
	}
	return b
}

// Slug enforces a URL-friendly identifier (lowercase, digits, hyphens).
func (b *Builder) Slug(field, value string) *Builder {
	if !slugRegex.MatchString(value) {
		b.err.WithDetail(field, "must be a valid slug (lowercase alphanumerics and dashes)")
	}
	return b
}

// OneOf reports an error if value is not contained in allowed.
func (b *Builder) OneOf(field, value string, allowed ...string) *Builder {
	for _, a := range allowed {
		if a == value {
			return b
		}
	}
	b.err.WithDetail(field, "must be one of "+strings.Join(allowed, ", "))
	return b
}

// Done returns an error iff at least one detail was recorded.
func (b *Builder) Done() error {
	if b.err == nil || len(b.err.Details) == 0 {
		return nil
	}
	return b.err
}
