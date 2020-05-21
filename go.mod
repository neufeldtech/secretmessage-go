module github.com/neufeldtech/smsg-go

// +heroku goVersion go1.13
go 1.13

require (
	github.com/alicebob/miniredis/v2 v2.11.4
	github.com/gin-gonic/gin v1.5.0
	github.com/go-redis/redis v6.15.7+incompatible
	github.com/gomodule/redigo v2.0.0+incompatible // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/lithammer/shortuuid v3.0.0+incompatible
	github.com/onsi/ginkgo v1.10.1 // indirect
	github.com/onsi/gomega v1.7.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/common v0.9.1
	github.com/slack-go/slack v0.6.4
	github.com/stretchr/testify v1.4.0
	go.elastic.co/apm v1.8.0
	go.elastic.co/apm/module/apmgin v1.8.0
	go.elastic.co/apm/module/apmgoredis v1.8.0
	go.elastic.co/apm/module/apmhttp v1.8.0
	golang.org/x/net v0.0.0-20190923162816-aa69164e4478
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
)
