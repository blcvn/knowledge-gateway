module github.com/vnp-community/vnp-memory/apps/memory

go 1.25.0

require (
	github.com/nats-io/nats-server/v2 v2.11.4
	github.com/nats-io/nats.go v1.52.0
	github.com/spf13/viper v1.18.2
	github.com/vnp-community/vnp-memory/gateway v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.81.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pelletier/go-toml/v2 v2.1.0 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	vnp-memory/services/graphiti-store v0.0.0-00010101000000-000000000000 // indirect
	vnp-memory/services/vnp-search-hub v0.0.0-00010101000000-000000000000 // indirect
)

require (
	github.com/google/go-tpm v0.9.5 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/minio/highwayhash v1.0.3 // indirect
	github.com/nats-io/jwt/v2 v2.7.4 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/vnp-community/vnp-memory/shared/pkg/forward v0.0.0-00010101000000-000000000000
	github.com/vnp-community/vnp-memory/services/vnp-dashboard v0.0.0-00010101000000-000000000000
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/time v0.11.0 // indirect
)

replace github.com/vnp-community/vnp-memory/shared/pkg/forward => ../../shared/pkg/forward

replace github.com/vnp-community/vnp-memory/services/vnp-dashboard => ../../services/vnp-dashboard

replace github.com/vnp-community/vnp-memory/gateway => ../../gateway

replace vnp-memory/services/vnp-search-hub => ../../services/vnp-search-hub

replace vnp-memory/services/graphiti-store => ../../services/graphiti-store
