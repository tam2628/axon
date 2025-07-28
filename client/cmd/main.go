package main

import "github.com/tam2628/axon/internal/common"

func main() {
	app := common.InitApp()
	app.RunServerWithGracefulShutdown(8080)
}
