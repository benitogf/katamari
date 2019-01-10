# SAMO

[![Build Status][build-image]][build-url]


[build-url]: https://travis-ci.com/benitogf/samo
[build-image]: https://api.travis-ci.com/benitogf/samo.svg?token=b628aVyTMNXTpbUCmJtn&branch=master&style=flat-square

pub/sub and http service for leveldb object storage

| method | description | url    |
| ------------- |:-------------:| -----:|
| get | key list | http://{host}:{port} |

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
