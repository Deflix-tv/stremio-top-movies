module github.com/doingodswork/stremio-top-movies

go 1.14

require (
	github.com/doingodswork/deflix-stremio v0.4.0
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.7.4
)

replace github.com/doingodswork/deflix-stremio => ../deflix-stremio
