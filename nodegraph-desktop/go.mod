module github.com/latentarts/gu-examples/nodegraph-desktop

go 1.26.2

require (
	github.com/latentart/gu v0.0.0
	github.com/webview/webview_go v0.0.0-20240831120633-6173450d4dd6
)

replace github.com/latentart/gu => ../../gu

replace github.com/webview/webview_go => ./third_party/webview_go
