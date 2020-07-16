package katamari_test

import katamari "bitbucket.org/idxgames/auth"

func ExampleServer() {
	app := katamari.Server{}
	app.Start("localhost:8800")
	app.WaitClose()
}
