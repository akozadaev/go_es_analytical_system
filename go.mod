module github.com/akozadaev/go_es_analytical_system

go 1.25.5

require (
	github.com/akozadaev/go_readiness v1.0.0
	github.com/elastic/go-elasticsearch/v8 v8.19.0
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/gorilla/mux v1.8.1
	github.com/lib/pq v1.10.9
	github.com/swaggo/http-swagger v1.3.4
	github.com/swaggo/swag v1.16.6
	ollamaclient v0.0.0-20260327221050-643afc4eaf17
)

require (
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/elastic/elastic-transport-go/v8 v8.7.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/spec v0.20.6 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/sashabaranov/go-openai v1.41.2 // indirect
	github.com/swaggo/files v0.0.0-20220610200504-28940afbdbfe // indirect
	go.opentelemetry.io/otel v1.28.0 // indirect
	go.opentelemetry.io/otel/metric v1.28.0 // indirect
	go.opentelemetry.io/otel/trace v1.28.0 // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace ollamaclient v0.0.0-20260327221050-643afc4eaf17 => github.com/akozadaev/go_ollama_client v0.0.0-20260327221050-643afc4eaf17
