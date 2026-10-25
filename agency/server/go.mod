module cherrypi.vladfaust.com/agency

go 1.26.4

require cherrypi.vladfaust.com/shared v0.0.0

replace cherrypi.vladfaust.com/shared => ../../shared

require (
	connectrpc.com/connect v1.20.0
	connectrpc.com/cors v0.1.0
	github.com/aws/aws-sdk-go-v2 v1.47.3
	github.com/aws/aws-sdk-go-v2/config v1.33.9
	github.com/aws/aws-sdk-go-v2/credentials v1.20.9
	github.com/aws/aws-sdk-go-v2/service/s3 v1.114.3
	github.com/coder/websocket v1.8.15
	github.com/coreos/go-oidc/v3 v3.21.0
	github.com/go-ldap/ldap/v3 v3.4.15
	github.com/google/uuid v1.6.0
	github.com/hashicorp/yamux v0.1.2
	github.com/jackc/pgx/v5 v5.11.0
	github.com/joho/godotenv v1.5.1
	github.com/pressly/goose/v3 v3.28.0
	github.com/rs/cors v1.11.1
	golang.org/x/crypto v0.57.0
	golang.org/x/oauth2 v0.37.0
	google.golang.org/protobuf v1.36.12
	modernc.org/sqlite v1.60.1
)

require (
	github.com/Azure/go-ntlmssp v0.1.1 
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.22 
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.20.3 
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.6 
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.6 
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.5.6 
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.21 
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.11.7 
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.14.6 
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.20.6 
	github.com/aws/aws-sdk-go-v2/service/signin v1.10.4 
	github.com/aws/aws-sdk-go-v2/service/sso v1.38.4 
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.43.4 
	github.com/aws/aws-sdk-go-v2/service/sts v1.51.4 
	github.com/aws/smithy-go v1.28.5 
	github.com/dustin/go-humanize v1.0.1 
	github.com/go-asn1-ber/asn1-ber v1.5.8 
	github.com/go-jose/go-jose/v4 v4.1.4 
	github.com/jackc/pgpassfile v1.0.0 
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 
	github.com/jackc/puddle/v2 v2.2.2 
	github.com/mattn/go-isatty v0.0.24 
	github.com/mfridman/interpolate v0.0.2 
	github.com/ncruces/go-strftime v1.0.0 
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec 
	github.com/sethvargo/go-retry v0.4.0 
	go.uber.org/multierr v1.11.0 
	golang.org/x/mod v0.41.0 
	golang.org/x/sync v0.23.0 
	golang.org/x/sys v0.48.0 
	golang.org/x/text v0.42.0 
	modernc.org/libc v1.77.1 
	modernc.org/mathutil v1.7.1 
	modernc.org/memory v1.12.1 
)
