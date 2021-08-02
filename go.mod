module blog-log-agent

go 1.16

require (
	github.com/ThalesIgnite/crypto11 v1.2.1
	github.com/pelletier/go-toml v1.8.0
)

replace github.com/ThalesIgnite/crypto11 => github.com/doanac/crypto11 v1.2.2-0.20200715151421-f3d2e17ac497

replace github.com/pelletier/go-toml => github.com/foundriesio/go-toml v1.8.1-0.20200721033514-2232fec316b9
