package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"serviceA/model"
)

type TemperatureResponse struct {
	City       string  `json:"city"`
	Celsius    float64 `json:"temp_C"`
	Fahrenheit float64 `json:"temp_F"`
	Kelvin     float64 `json:"temp_K"`
}

func main() {
	http.HandleFunc("/", handleRequest)
	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	fmt.Printf("Listening on port %s\n", port)
	http.ListenAndServe(":"+port, nil)
}

// isValidZipCode checks if the given zipCode is a valid 8-digit number.
func isValidZipCode(zipCode string) bool {
	return regexp.MustCompile(`^\d{8}$`).MatchString(zipCode)
}

func sendRequestToB(cep string) (TemperatureResponse, error, int) {
	serviceBURL := os.Getenv("SERVICE_B_URL")
	if serviceBURL == "" {
		serviceBURL = "http://localhost:8081"
	}

	url := fmt.Sprintf("%s?cep=%s", serviceBURL, cep)

	resp, err := http.Get(url)
	if err != nil {
		return TemperatureResponse{}, fmt.Errorf("error sending request to Service B: %w", err), http.StatusInternalServerError
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResponse struct {
			Message string `json:"message"`
		}
		json.NewDecoder(resp.Body).Decode(&errorResponse)
		return TemperatureResponse{}, fmt.Errorf(errorResponse.Message), resp.StatusCode
	}

	var tempResponse TemperatureResponse
	err = json.NewDecoder(resp.Body).Decode(&tempResponse)
	if err != nil {
		return TemperatureResponse{}, fmt.Errorf("error parsing response from Service B: %w", err), http.StatusInternalServerError
	}

	return tempResponse, nil, http.StatusOK

}

func handleRequest(responseWriter http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(responseWriter, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var requestInput model.CEPRequest

	err := json.NewDecoder(request.Body).Decode(&requestInput)
	if err != nil {
		http.Error(responseWriter, "Invalid Request", http.StatusBadRequest)
		return
	}

	responseWriter.Header().Add("Content-Type", "application/json")

	if !isValidZipCode(requestInput.CEP) {
		responseWriter.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(responseWriter).Encode(map[string]string{"message": "invalid zipcode"})
		return
	}

	temperatureResponse, err, status := sendRequestToB(requestInput.CEP)

	if err != nil {
		responseWriter.WriteHeader(status)
		json.NewEncoder(responseWriter).Encode(map[string]string{"message": err.Error()})
		return
	}

	responseWriter.WriteHeader(status)
	json.NewEncoder(responseWriter).Encode(temperatureResponse)

}
