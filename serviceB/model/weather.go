package model

import "strconv"

// WeatherResponse represents the response from a weather API.
// It contains the current temperature in Celsius.
type WeatherResponse struct {
	Current struct {
		TemperatureCelsius float64 `json:"temp_c"`
	} `json:"current"`
}

// TemperatureData represents temperature information in different scales.
// It uses Float64Marshal for custom JSON marshaling of float64 values.
type TemperatureData struct {
	City       string         `json:"city"`
	Celsius    Float64Marshal `json:"temp_C"`
	Fahrenheit Float64Marshal `json:"temp_F"`
	Kelvin     Float64Marshal `json:"temp_K"`
}

// Float64Marshal is a custom type for float64 values that implements
// custom JSON marshaling to avoid scientific notation.
type Float64Marshal float64

// MarshalJSON implements the json.Marshaler interface for Float64Marshal.
// It converts the float64 value to a string without scientific notation.
func (f Float64Marshal) MarshalJSON() ([]byte, error) {
	// Convert the float64 value to a string with full precision
	s := strconv.FormatFloat(float64(f), 'f', -1, 64)
	return []byte(s), nil
}
