module github.com/pandaye/days7go

go 1.17

require geerpc v0.0.0

replace (
	gee => ./geeweb
	geecache => ./geecache
	geerpc => ./geerpc
)
