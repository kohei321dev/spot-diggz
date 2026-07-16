package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kohei321dev/spot-diggz/internal/spot"
)

type SdzHandler struct {
	store spot.SdzStore
}

type healthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type errorResponse struct {
	Code      int    `json:"code"`
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
}

func (h SdzHandler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status:  "healthy",
		Message: "spotdiggz metadata api is running",
	})
}

func (h SdzHandler) createSpot(w http.ResponseWriter, r *http.Request) {
	var input spot.SdzCreateSpotInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}
	created, err := h.store.Create(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h SdzHandler) listSpots(w http.ResponseWriter, r *http.Request) {
	filter, err := listFilterFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	spots, err := h.store.List(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, spots)
}

func (h SdzHandler) listSpotsGeoJSON(w http.ResponseWriter, r *http.Request) {
	filter, err := listFilterFromRequest(r)
	if err != nil {
		writeError(w, err)
		return
	}
	spots, err := h.store.List(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toFeatureCollection(spots))
}

func (h SdzHandler) getSpot(w http.ResponseWriter, r *http.Request) {
	found, err := h.store.Get(r.Context(), chi.URLParam(r, "spotID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, found)
}

func (h SdzHandler) updateSpot(w http.ResponseWriter, r *http.Request) {
	var input spot.SdzUpdateSpotInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}
	updated, err := h.store.Update(r.Context(), chi.URLParam(r, "spotID"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h SdzHandler) deleteSpot(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Delete(r.Context(), chi.URLParam(r, "spotID")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func listFilterFromRequest(r *http.Request) (spot.SdzListFilter, error) {
	query := r.URL.Query()
	bbox, err := spot.SdzParseBBox(query.Get("bbox"))
	if err != nil {
		return spot.SdzListFilter{}, err
	}
	visibility, err := spot.SdzParseVisibility(query.Get("visibility"))
	if err != nil {
		return spot.SdzListFilter{}, err
	}
	return spot.SdzListFilter{
		SdzBBox:       bbox,
		Tags:          spot.SdzSplitTags(query.Get("tags")),
		SdzVisibility: visibility,
	}, nil
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return spot.SdzValidationError{Message: "invalid JSON body"}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	var validationError spot.SdzValidationError
	switch {
	case errors.As(err, &validationError):
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Code:      http.StatusBadRequest,
			ErrorCode: "SDZ-E-4001",
			Message:   validationError.Message,
		})
	case errors.Is(err, spot.SdzErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{
			Code:      http.StatusNotFound,
			ErrorCode: "SDZ-E-4041",
			Message:   "spot not found",
		})
	default:
		writeJSON(w, http.StatusInternalServerError, errorResponse{
			Code:      http.StatusInternalServerError,
			ErrorCode: "SDZ-E-5001",
			Message:   "internal server error",
		})
	}
}
