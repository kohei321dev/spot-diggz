package httpapi

import "github.com/kohei321dev/spot-diggz/internal/spot"

type featureCollection struct {
	Type     string    `json:"type"`
	Features []feature `json:"features"`
}

type feature struct {
	Type       string     `json:"type"`
	Geometry   geometry   `json:"geometry"`
	Properties properties `json:"properties"`
}

type geometry struct {
	Type        string     `json:"type"`
	Coordinates [2]float64 `json:"coordinates"`
}

type properties struct {
	SdzSpotID     string             `json:"spotId"`
	Name          string             `json:"name"`
	Description   string             `json:"description,omitempty"`
	Tags          []string           `json:"tags"`
	SdzVisibility spot.SdzVisibility `json:"visibility"`
	CreatedAt     string             `json:"createdAt"`
	UpdatedAt     string             `json:"updatedAt"`
}

func toFeatureCollection(spots []spot.SdzSpot) featureCollection {
	features := make([]feature, 0, len(spots))
	for _, item := range spots {
		features = append(features, feature{
			Type: "Feature",
			Geometry: geometry{
				Type:        "Point",
				Coordinates: [2]float64{item.SdzLocation.Lng, item.SdzLocation.Lat},
			},
			Properties: properties{
				SdzSpotID:     item.SdzSpotID,
				Name:          item.Name,
				Description:   item.Description,
				Tags:          item.Tags,
				SdzVisibility: item.SdzVisibility,
				CreatedAt:     item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:     item.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			},
		})
	}
	return featureCollection{
		Type:     "FeatureCollection",
		Features: features,
	}
}
