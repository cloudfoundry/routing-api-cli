[![Build Status](https://travis-ci.org/cloudfoundry-incubator/routing-api-cli.svg)](https://travis-ci.org/cloudfoundry-incubator/routing-api-cli)

# Routing API CLI

The Routing API CLI lets you list, register, and unregister routes with the Cloud Foundry [Routing API](https://github.com/cloudfoundry-incubator/routing-api).

## Dependencies

The Routing API uses OAuth tokens to authenticate clients. To obtain a token from UAA an OAuth client must first be created for the API client in UAA. For instructions on registering OAuth clients, see [Routing API Server Configuration](https://github.com/cloudfoundry-incubator/routing-api#oauth-clients).

## Installation

### Download Binaries

See [Releases](https://github.com/cloudfoundry-incubator/routing-api-cli/releases)

### Compile

- Go should be installed and in the PATH
- GOPATH should be set as described in http://golang.org/doc/code.html
- [godep](https://github.com/tools/godep) installed and in the PATH

- Build the binary, and place it in your gopath:

  ```bash
  go get github.com/cloudfoundry-incubator/routing-api-cli
  godep restore
  go build -o $GOPATH/bin/rtr
  ```

## Usage

Each command has required arguments and route structure.

Required arguments:

**--api**: the routing API endpoint, e.g. https://routing-api.example.com<br />
**--client-id**: the id of the client registered with your OAuth provider with the [proper authorities](https://github.com/cloudfoundry-incubator/routing-api#authorization-token), e.g. admin<br />
**--client-secret**: your OAuth client secret, e.g. admin-secret<br />
**--oauth-url**: the OAuth provider endpoint with optional port, e.g. https://uaa.example.com

Routes are described as JSON: `'[{"route":"foo.com","port":65340,"ip":"1.2.3.4","ttl":60, "route_service_url":"https://route-service.example.cf-app.com"}]'`

### List Routes
```bash
rtr list [args]
```

### Register Route(s)
```bash
rtr register [args] [routes]
```

### Unregister Route(s)
```bash
rtr unregister [args] [routes]
```

### Tracing Requests and Responses

By specifying the environment variable `RTR_TRACE=true`, `rtr` will output the HTTP requests and responses that it makes and receives.
```bash
export RTR_TRACE=true
rtr list [args]
```

Notes:
- Route "ttl" definition is ignored for unregister.
- CLI will appear successful when unregistering routes that do not exist.
- The `route_service_url` is an optional value, and must be a HTTPS url.

###Examples

```bash
rtr list --api https://routing-api.example.com --client-id admin --client-secret admin-secret --oauth-url https://uaa.example.com

rtr register --api https://routing-api.example.com --client-id admin --client-secret admin-secret --oauth-url https://uaa.example.com '[{"route":"mynewroute.com","port":12345,"ip":"1.2.3.4","ttl":60}]'

rtr unregister --api https://routing-api.example.com --client-id admin --client-secret admin-secret --oauth-url https://uaa.example.com '[{"route":"undesiredroute.com","port":12345,"ip":"1.2.3.4"}]'
```
