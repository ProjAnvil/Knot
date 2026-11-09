module github.com/ProjAnvil/knot/tools

go 1.25.4

// Dev-time replace, same ruling as backend/go.mod: the models module is not
// published yet, so an unpublished version cannot resolve through the proxy.
replace github.com/ProjAnvil/knot/models => ../models

require (
	github.com/ProjAnvil/knot/models v1.0.0
	github.com/gzuidhof/tygo v0.2.21
	github.com/invopop/jsonschema v0.14.0
)

require (
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.2 // indirect
	github.com/fatih/structtag v1.2.0 // indirect
	github.com/pb33f/ordered-map/v2 v2.3.1 // indirect
	go.yaml.in/yaml/v4 v4.0.0-rc.2 // indirect
	golang.org/x/mod v0.5.1 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/tools v0.1.9 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)
