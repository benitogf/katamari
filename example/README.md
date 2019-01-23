# SAMO - client example

To try the example:

## run the service:

with [golang installed](https://golang.org/doc/install) from a terminal get the library:

```bash
go get github.com/benitogf/samo
```

then create a file `main.go` with the code:
```golang
package main

import (
	"github.com/benitogf/samo"
)

func main() {
	app := samo.Server{}
	app.Start("localhost:8800")
	app.WaitClose()
}
```

finally run the service with:
```bash
go run main.go
```


## run the client:

built with [create react app](https://reactjs.org/docs/create-a-new-react-app.html), from a terminal run:

```bash
git clone git@github.com:benitogf/samo.git
cd samo/example
npm install
npm start
```