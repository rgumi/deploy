package main

import (
	"depoy/gateway"
	_ "depoy/gateway"
	st "depoy/statemgt"
)

func main() {
	// load config

	// init gateway
	g := gateway.New(":8080")

	// start service
	st.Start(":8081", "", g)
}
