package katamari_test

import "github.com/benitogf/katamari"

func ExampleServer() {
	app := katamari.Server{}
	app.Start("localhost:8800")
	app.WaitClose()
}
