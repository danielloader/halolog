module github.com/go-gen-ecosystem/halolog/otelbridge

go 1.24.0

replace github.com/go-gen-ecosystem/halolog => ../

require (
	github.com/go-gen-ecosystem/halolog v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel/trace v1.37.0
)

require (
	go.opentelemetry.io/otel v1.37.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
