// Package httpx contains HTTP-layer concerns shared by every module:
// router setup, middleware, and standardised response helpers.
//
// All public APIs return JSON with a stable envelope so the dashboard can
// branch deterministically on success / failure shapes.
package httpx

import (
	"encoding/json"
	"errors"
	"net/http"

	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
)

// Response is the canonical success envelope. Modules wrap their DTOs in
// Data; transport metadata (request id, pagination cursors, timing) goes
// into Meta.
type Response struct {
	Data interface{}            `json:"data,omitempty"`
	Meta map[string]interface{} `json:"meta,omitempty"`
}

// ErrorBody is the canonical error envelope.
type ErrorBody struct {
	Error ErrorPayload `json:"error"`
}

// ErrorPayload is the body of an error response.
type ErrorPayload struct {
	Kind    platformerrors.Kind `json:"kind"`
	Code    string              `json:"code,omitempty"`
	Message string              `json:"message"`
	Details map[string]string   `json:"details,omitempty"`
}

// JSON writes a successful JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

// OK writes a 200 with the supplied data wrapped in the canonical envelope.
func OK(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, Response{Data: data})
}

// Created writes a 201.
func Created(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, Response{Data: data})
}

// NoContent writes a 204.
func NoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

// Error inspects err and writes the appropriate status + envelope.
//
// Usage: handlers should always call this rather than crafting responses by
// hand so error shapes stay uniform and discoverable.
func Error(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	var pe *platformerrors.Error
	if !errors.As(err, &pe) {
		pe = platformerrors.Internal("internal error").WithCause(err)
	}

	JSON(w, statusFor(pe.Kind), ErrorBody{
		Error: ErrorPayload{
			Kind:    pe.Kind,
			Code:    pe.Code,
			Message: pe.Message,
			Details: pe.Details,
		},
	})
}

func statusFor(kind platformerrors.Kind) int {
	switch kind {
	case platformerrors.KindValidation:
		return http.StatusUnprocessableEntity
	case platformerrors.KindUnauthorized:
		return http.StatusUnauthorized
	case platformerrors.KindForbidden:
		return http.StatusForbidden
	case platformerrors.KindNotFound:
		return http.StatusNotFound
	case platformerrors.KindConflict:
		return http.StatusConflict
	case platformerrors.KindRateLimited:
		return http.StatusTooManyRequests
	case platformerrors.KindUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// DecodeJSON reads a JSON body into dst with sensible defaults (size limit,
// disallow unknown fields). Returns a *platformerrors.Error of Kind
// validation on malformed input.
func DecodeJSON(r *http.Request, dst interface{}) error {
	const maxBody = 1 << 20 // 1 MiB
	r.Body = http.MaxBytesReader(nil, r.Body, maxBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return platformerrors.Validation("invalid JSON body").WithCause(err)
	}
	return nil
}
