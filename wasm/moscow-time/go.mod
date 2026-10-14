module github.com/moscow_time

go 1.24

require github.com/spinframework/spin-go-sdk/v2 v2.2.1

require github.com/julienschmidt/httprouter v1.3.0 // indirect

replace github.com/spinframework/spin-go-sdk/v2 => ./vendor-debug/spin-go-sdk
