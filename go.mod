module github.com/pandaye/days7go

go 1.17

require geerpc v0.0.0

require google.golang.org/protobuf v1.28.0 // indirect

replace (
	gee => ./geeweb
	geecache => ./geecache
	geerpc => ./geerpc
)
