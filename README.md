# QUIC Pub Sub

## About

This project contains 3 services that communicate via QUIC:
1. Server (in `/server`)
1. Publisher (in `/publisher`)
1. Subscriber (in `/subscriber`)

* Publishers and Subscribers connect to the Server
* Publishers send periodic messages to the Server if and only if there are subscribers connected to the Server
* The Server sends out messages from Publishers to all connected Subscribers.

## Running the apps

Generate TLS certificates before the first run:
```shell
bash gen_certs.sh
```

Build:
```shell
go build -o bin/server ./server
go build -o bin/publisher ./publisher
go build -o bin/subscriber ./subscriber
```

From the project root:
1. build the server
```shell
./bin/server
```
2. open publishers as subscribers from different shells:
```shell
./bin/publisher
```
```shell
./bin/subscriber
```

If the commands complain, run them with `-help` to see how to modify parameters.

## Notes

### Relationships

In each app:
- Package `transport` contains code for remote communication and passes data onto the `app` package if one exists,
- Package `app` contains the logical part of the server, excluding any data transport/RPC specifics.

### Directories "internal"

Packages `internal` might not be necessary but they are here to signify (and enforce by the compiler) that code is not shared between each of the 3 services, as they all live in the same project. The code that _is_ shared lives in `pkg`.

### Loggers

Loggers are passed to constructors as arguments, not via [Options](https://github.com/uber-go/guide/blob/master/style.md#functional-options), in order to just save some time.

## Testing

```shell
go test ./...
```

Generate mocks if they need to change
```shell
go generate ./...
```
