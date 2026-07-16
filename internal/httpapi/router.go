package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kohei321dev/spot-diggz/internal/spot"
)

func NewSdzRouter(store spot.SdzStore) http.Handler {
	handler := SdzHandler{store: store}

	router := chi.NewRouter()
	router.Get("/sdz/health", handler.health)
	router.Get("/sdz/spots", handler.listSpots)
	router.Post("/sdz/spots", handler.createSpot)
	router.Get("/sdz/spots.geojson", handler.listSpotsGeoJSON)
	router.Get("/sdz/spots/{spotID}", handler.getSpot)
	router.Patch("/sdz/spots/{spotID}", handler.updateSpot)
	router.Delete("/sdz/spots/{spotID}", handler.deleteSpot)
	return router
}
