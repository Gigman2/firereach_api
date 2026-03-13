package station

import (
	"context"
	"math"
	"sort"

	"github.com/firereach/api/internal/domain"
)

type ListNearestStations struct {
	repo domain.StationRepository
}

func NewListNearestStations(r domain.StationRepository) *ListNearestStations {
	return &ListNearestStations{repo: r}
}

func (uc *ListNearestStations) Execute(ctx context.Context, lat, lng float64, limit int) ([]domain.Station, error) {
	stations, err := uc.repo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	return sortByDistance(stations, lat, lng, limit), nil
}

func sortByDistance(stations []domain.Station, lat, lng float64, limit int) []domain.Station {
	sort.Slice(stations, func(i, j int) bool {
		di := haversine(lat, lng, stations[i].Lat, stations[i].Lng)
		dj := haversine(lat, lng, stations[j].Lat, stations[j].Lng)
		return di < dj
	})
	if limit > 0 && limit < len(stations) {
		stations = stations[:limit]
	}
	return stations
}

func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0
	dLat := toRadians(lat2 - lat1)
	dLng := toRadians(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRadians(lat1))*math.Cos(toRadians(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

func toRadians(deg float64) float64 {
	return deg * math.Pi / 180
}
