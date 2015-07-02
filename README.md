[![Build Status](https://travis-ci.org/cloudfoundry-incubator/routing-api-cli.svg)](https://travis-ci.org/cloudfoundry-incubator/routing-api-cli)

# Routing API CLI

The Routing API CLI lets you list, register, and unregister routes with the Cloud Foundry [Routing API](https://github.com/cloudfoundry-incubator/routing-api).

## Installation

### Download Binaries

See [Releases](https://github.com/cloudfoundry-incubator/routing-api-cli/releases)

### Compile

1. Install [godep](https://github.com/tools/godep)
- Build the binary, and place it in your gopath:

  ```bash
  godep restore
  go build -o rtr
  cp rtr $GOPATH/bin/
  ```

## Usage

Each command has required arguments and route structure.

Required arguments:

**--api**: the routing API endpoint, e.g. https://routing-api.example.com<br />
**--oauth-name**: the name of the client registered with your OAuth provider with the [proper authorities](https://github.com/cloudfoundry-incubator/routing-api#authorization-token), e.g. admin<br />
**--oauth-password**: your OAuth password, e.g. admin-secret<br />
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
rtr list --api https://routing-api.example.com --oauth-name admin --oauth-password admin-secret --oauth-url https://uaa.example.com

rtr register --api https://routing-api.example.com --oauth-name admin --oauth-password admin-secret --oauth-url https://uaa.example.com '[{"route":"mynewroute.com","port":12345,"ip":"1.2.3.4","ttl":60}]'

rtr unregister --api https://routing-api.example.com --oauth-name admin --oauth-password admin-secret --oauth-url https://uaa.example.com '[{"route":"undesiredroute.com","port":12345,"ip":"1.2.3.4"}]'
```
