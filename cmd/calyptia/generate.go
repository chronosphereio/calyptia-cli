package main

//go:generate go run -mod=mod github.com/rjeczalik/interfaces/cmd/interfacer -for github.com/calyptia/api/client.Client -as main.Client -o ./client_gen.go
//go:generate go run -mod=mod github.com/matryer/moq -rm -stub -out client_mock_gen.go . Client
