package api

type NeoFeedParams struct {
	StartDate string `json:"start_date,omitempty" form:"start_date"` // Format: YYYY-MM-DD
	EndDate   string `json:"end_date,omitempty"   form:"end_date"`   // Format: YYYY-MM-DD (Defaults to 7 days after StartDate)
	APIKey    string `json:"api_key,omitempty"    form:"api_key"`
}

type NeoLookupParams struct {
	AsteroidID string `json:"asteroid_id" form:"asteroid_id" uri:"id"`
	APIKey     string `json:"api_key,omitempty" form:"api_key"`
}

func (p *NeoFeedParams) ToQueryMap() map[string]string {
	params := make(map[string]string)
	apiKey := p.APIKey
	if apiKey == "" {
		apiKey = "DEMO_KEY"
	}
	params["api_key"] = apiKey

	if p.StartDate != "" {
		params["start_date"] = p.StartDate
	}
	if p.EndDate != "" {
		params["end_date"] = p.EndDate
	}
	return params
}

type DistanceUnit struct {
	EstimatedDiameterMin float64 `json:"estimated_diameter_min"`
	EstimatedDiameterMax float64 `json:"estimated_diameter_max"`
}

type EstimatedDiameter struct {
	Kilometers DistanceUnit `json:"kilometers"`
	Meters     DistanceUnit `json:"meters"`
	Miles      DistanceUnit `json:"miles"`
	Feet       DistanceUnit `json:"feet"`
}

type RelativeVelocity struct {
	KilometersPerSecond string `json:"kilometers_per_second"`
	KilometersPerHour   string `json:"kilometers_per_hour"`
	MilesPerHour        string `json:"miles_per_hour"`
}

type MissDistance struct {
	Astronomical string `json:"astronomical"`
	Lunar        string `json:"lunar"`
	Kilometers   string `json:"kilometers"`
	Miles        string `json:"miles"`
}

type CloseApproachData struct {
	CloseApproachDate      string           `json:"close_approach_date"`
	CloseApproachDateFull  string           `json:"close_approach_date_full"`
	EpochDateCloseApproach int64            `json:"epoch_date_close_approach"`
	RelativeVelocity       RelativeVelocity `json:"relative_velocity"`
	MissDistance           MissDistance     `json:"miss_distance"`
	OrbitingBody           string           `json:"orbiting_body"`
}

type NearEarthObject struct {
	ID                             string              `json:"id"`
	NeoReferenceID                 string              `json:"neo_reference_id"`
	Name                           string              `json:"name"`
	NasaJplURL                     string              `json:"nasa_jpl_url"`
	AbsoluteMagnitudeH             float64             `json:"absolute_magnitude_h"`
	EstimatedDiameter              EstimatedDiameter   `json:"estimated_diameter"`
	IsPotentiallyHazardousAsteroid bool                `json:"is_potentially_hazardous_asteroid"`
	CloseApproachData              []CloseApproachData `json:"close_approach_data"`
	IsSentryObject                 bool                `json:"is_sentry_object"`
}

type NeoFeedResponse struct {
	ElementCount     int                          `json:"element_count"`
	NearEarthObjects map[string][]NearEarthObject `json:"near_earth_objects"` // Keyed by date (YYYY-MM-DD)
}
