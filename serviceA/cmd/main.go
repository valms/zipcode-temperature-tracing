package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
	"log"
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

var tracer trace.Tracer

func main() {
	ctx := context.Background()
	tp, err := initTracer("service-a")

	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = tp.Shutdown(ctx) }()

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	tracer = tp.Tracer("service-a")

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

func sendRequestToB(ctx context.Context, cep string) (TemperatureResponse, error, int) {
	ctx, span := tracer.Start(ctx, "sendRequestToB", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	span.SetAttributes(
		attribute.String("cep", cep),
		attribute.String("service", "ServiceA"),
	)

	serviceBURL := os.Getenv("SERVICE_B_URL")
	if serviceBURL == "" {
		serviceBURL = "http://localhost:8081"
	}

	url := fmt.Sprintf("%s?cep=%s", serviceBURL, cep)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

	if err != nil {
		span.SetStatus(codes.Error, "error creating request")
		return TemperatureResponse{}, fmt.Errorf("error creating request: %w", err), http.StatusInternalServerError
	}

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	span.AddEvent("Sending request to Service B")
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		span.SetStatus(codes.Error, "error sending request to Service B")
		return TemperatureResponse{}, fmt.Errorf("error sending request to Service B: %w", err), http.StatusInternalServerError
	}
	defer resp.Body.Close()

	span.AddEvent("Received response from Service B")

	if resp.StatusCode != http.StatusOK {
		var errorResponse struct {
			Message string `json:"message"`
		}
		json.NewDecoder(resp.Body).Decode(&errorResponse)
		span.SetStatus(codes.Error, errorResponse.Message)
		return TemperatureResponse{}, fmt.Errorf(errorResponse.Message), resp.StatusCode
	}

	var tempResponse TemperatureResponse
	err = json.NewDecoder(resp.Body).Decode(&tempResponse)
	if err != nil {
		span.SetStatus(codes.Error, "error parsing response from Service B")
		return TemperatureResponse{}, fmt.Errorf("error parsing response from Service B: %w", err), http.StatusInternalServerError
	}

	span.SetStatus(codes.Ok, "Service B responded.")
	return tempResponse, nil, http.StatusOK

}

func handleRequest(responseWriter http.ResponseWriter, request *http.Request) {
	ctx, span := tracer.Start(request.Context(), "handleRequest-sa")
	defer span.End()

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

	span.SetAttributes(attribute.String("cep", requestInput.CEP))

	responseWriter.Header().Add("Content-Type", "application/json")

	if !isValidZipCode(requestInput.CEP) {
		span.SetStatus(codes.Error, "invalid zipcode")
		responseWriter.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(responseWriter).Encode(map[string]string{"message": "invalid zipcode"})
		return
	}

	temperatureResponse, err, status := sendRequestToB(ctx, requestInput.CEP)

	if err != nil {
		responseWriter.WriteHeader(status)
		span.SetStatus(codes.Error, err.Error())
		json.NewEncoder(responseWriter).Encode(map[string]string{"message": err.Error()})
		return
	}

	span.SetStatus(codes.Ok, "Completed")
	responseWriter.WriteHeader(status)
	json.NewEncoder(responseWriter).Encode(temperatureResponse)

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
