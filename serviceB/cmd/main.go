package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"serviceB/model"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func main() {
	ctx := context.Background()

	tp, err := initTracer("service-b")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = tp.Shutdown(ctx) }()

	tracer = tp.Tracer("service-b")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleRequest(w, r.WithContext(ctx))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Listening on port %s\n", port)
	http.ListenAndServe(":"+port, nil)
}

func isValidZipCode(zipCode string) bool {
	return regexp.MustCompile(`^\d{8}$`).MatchString(zipCode)
}

func fetchCityFromCEP(ctx context.Context, cep string) (string, error, int) {
	_, span := tracer.Start(ctx, "fetchCityFromCEP")
	defer span.End()

	if !isValidZipCode(cep) {
		span.SetStatus(codes.Error, "invalid zipcode")
		return "", errors.New("invalid zipcode"), http.StatusUnprocessableEntity
	}

	uri := fmt.Sprintf("https://viacep.com.br/ws/%s/json", cep)
	apiResponse, err, status := makeHTTPRequest[model.ZipCodeResponse](ctx, uri, http.MethodGet)

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return "", err, status
	}

	if apiResponse.City == "" || status == http.StatusNotFound {
		span.SetStatus(codes.Error, "can not find zipcode")
		return "", errors.New("can not find zipcode"), http.StatusNotFound
	}

	span.SetAttributes(attribute.String("city", apiResponse.City))
	return apiResponse.City, nil, status
}

func fetchWeather(ctx context.Context, city string) (float64, error, int) {
	_, span := tracer.Start(ctx, "fetchWeather")
	defer span.End()

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		span.SetStatus(codes.Error, "no API key set")
		return 0, errors.New("no API key set"), http.StatusBadRequest
	}

	encodedCity := url.QueryEscape(city)
	uri := fmt.Sprintf("https://api.weatherapi.com/v1/current.json?key=%s&q=%s&lang=pt", apiKey, encodedCity)
	apiResponse, err, status := makeHTTPRequest[model.WeatherResponse](ctx, uri, http.MethodGet)

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return 0, err, status
	}

	span.SetAttributes(attribute.Float64("temperature", apiResponse.Current.TemperatureCelsius))
	return apiResponse.Current.TemperatureCelsius, nil, status
}

func makeHTTPRequest[T any](ctx context.Context, uri string, method string) (T, error, int) {
	var result T

	req, err := http.NewRequestWithContext(ctx, method, uri, nil)
	if err != nil {
		return result, fmt.Errorf("error creating request: %w", err), http.StatusInternalServerError
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("error sending request: %w", err), http.StatusInternalServerError
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("unexpected status code: %d", resp.StatusCode), resp.StatusCode
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return result, fmt.Errorf("error parsing response: %w", err), http.StatusInternalServerError
	}

	return result, nil, http.StatusOK
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleRequest")
	defer span.End()

	cep := r.URL.Query().Get("cep")
	span.SetAttributes(attribute.String("cep", cep))

	city, err, status := fetchCityFromCEP(ctx, cep)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	temperature, err, status := fetchWeather(ctx, city)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	tempResponse := model.TemperatureData{
		City:       city,
		Celsius:    model.Float64Marshal(temperature),
		Fahrenheit: model.Float64Marshal(temperature*1.8 + 32),
		Kelvin:     model.Float64Marshal(temperature + 273),
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tempResponse)
}

func initTracer(serviceName string) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracehttp.New(context.Background())
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}
