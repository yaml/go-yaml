module example.com/json-comments-go-run

go 1.24

require (
	go.yaml.in/yaml/v4 v4.0.0-rc.6
	go.yaml.in/yaml/v4/plugin/jsoncomments v0.0.0
)

require (
	bitbucket.org/pcastools/hash v1.0.5 // indirect
	github.com/gloathub/go-readline v0.1.2 // indirect
	github.com/gloathub/gojava v0.1.0 // indirect
	github.com/glojurelang/glojure v0.7.15 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/yamlstar/yamlstar-plugin-json-comments v0.1.9 // indirect
	go4.org/intern v0.0.0-20220617035311-6925f38cc365 // indirect
	go4.org/unsafe/assume-no-moving-gc v0.0.0-20230525183740-e7c30c78aeb2 // indirect
	golang.org/x/sys v0.8.0 // indirect
)

replace go.yaml.in/yaml/v4 => ../..

replace go.yaml.in/yaml/v4/plugin/jsoncomments => ../../plugin/jsoncomments
