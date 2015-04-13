#Routing API CLI

-------

The Routing API CLI exists to make the Routing API easily consumable via a CLI

Currently, the Routing API CLI supports registering and unregistering routes with a Routing API

```bash
./rtr register [args] [routes]
```
where `[args]` are
```
-api [the routing api endpoint]
-oauth-name [your oauth client name]
-oauth-password [your oauth password or secret]
-oauth-url [the oauth provider endpoint]
-oauth-port [ the oauth provider port]
```
All of these arguments are required.

-------

Routes take the form of an array of json encoded route endpoints
```
'[{"route":"foo.com","port":65340,"ip":"1.2.3.4","ttl":5,"log_guid":"foo-guid"}]'
```
