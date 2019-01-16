# SAMO

[![Build Status][build-image]][build-url]


[build-url]: https://travis-ci.org/benitogf/samo
[build-image]: https://api.travis-ci.org/benitogf/samo.svg?branch=master&style=flat-square

pub/sub and http service for leveldb object storage

| method | description | url    |
| ------------- |:-------------:| -----:|
| get | key list | http://{host}:{port} |

#### time

| method | description | url    |
| ------------- |:-------------:| -----:|
| websocket| time tick | ws://{host}:{port}/time |


#### single allocation (SA)

| method | description | url    |
| ------------- |:-------------:| -----:|
| websocket| key data events: update, delete | ws://{host}:{port}/sa/{key} |
| POST | create/update | http://{host}:{port}/r/sa |
| GET | get | http://{host}:{port}/r/sa/{key} |

#### multiple objects (MO)

| method  | description | url    |
| ------------- |:-------------:| -----:|
| websocket | key data events: new, update, delete | ws://{host}:{port}/mo/{key} |
| POST | create/update | http://{host}:{port}/r/mo |
| GET | get | http://{host}:{port}/r/mo/{key} |


## SA subscription events

#### get (sent after handshake and on each new/update event)
---
```js
{
    created: 0,
    updated: 0,
    index: '',
    data: 'e30='
}
```

#### put
---
```js
{
    data: 'e30='
}
```

## MO subscription events

#### get (sent after handshake)
---
```js
[
    {
        created: 1546660572033308700,
        updated: 0,
        index: '1576d7988025d81c0',
        data: 'e30='
    }
    ...
]
```

#### put
---
```js
{
    index: 'test',
    data: 'e30='
}
```

### del
---
```js
{
    op: 'DEL',
    index: 'test'
}
```

## Data archetypes

Define custom acceptance criteria of data using key glob patterns

```go
package main

import "github.com/benitogf/samo"

func main() {
	app := samo.Server{}
	app.Archetypes = samo.Archetypes{
		"things/*": func(data string) bool {
			return data == "object"
		},
		"bag": func(data string) bool {
			return data == "marbles"
		},
	}
	app.Start("localhost:9889", "test/db", "/")
	app.WaitClose()
}
```
