go-troubleshroute
========

A simple Go-based http(s) responder for testing OCP route termination configurations.

Deployment
----------

Run a local instance:

    $ export HTTP_PORT=<port>   # sudoless
    $ export HTTPS_PORT=<port>  # sudoless 

    $ make run

or in a container:

    $ make run-container

or for more flexibility:

    $ podman run --rm --volume $(pwd)/pki/:/usr/src/app/pki:Z --publish <LOCAL_PORT>:443 quay.io/bverschueren/go-troubleshroute:latest


Run as a deployment:

 - exposed over a reencrypt route:

    *if previous pki exists first run `make clean-pki`*

    ```
    $ CERT_CN=go-troubleshroute-<namespace>.apps.example.com
    $ make ocp-reencrypt
     <...>
    $ $ curl -k https://go-troubleshroute-<namespace>.apps.example.com
    Got / request from Connecting from xx.xx.xx.xx, forwarded by
    yy.yy.yy.yy:yyyy!
    ```

 - exposed over a passthrough route:

    *if previous pki exists first run `make clean-pki`*

    ```
    $ CERT_CN=go-troubleshroute-<namespace>.apps.example.com
    $ make ocp-passthrough
    <...>
    $ curl --cacert pki/cacert.pem
    https://go-troubleshroute-<namespace>.apps.example.com
    Got / request from Connecting from xx.xx.xx.xx:xxxxx
    ```

Configuration
-------------

By default, the "server" listens on `*:${HTTP_PORT}` + `*:${HTTPS_PORT}` offering `${TLS_CERT}` protected by key file `${TLS_CERT}` with defaults:


| ENV | Default | Use |
| --- | --- | --- |
|HTTPS_PORT | 443       | HTTPS listen port |
|HTTP_PORT | 80       | HTTPS listen port |
|TLS_CERT    | /usr/src/app/pki/tls.crt | Default TLS cert offered for HTTPS. This may be a wildcard used as a default/fallback if no SNI is offered.|
|TLS_KEY     | /usr/src/app/pki//tls.key | TLS key |
|SERVER_TLS_CERT | <empty> | This is a "final/server" TLS cert offered if SNI used|
|SERVER_TLS_KEY | <empty> | TLS key |

## How is this useful?

Some examples to troubelshoot/verify routes:

* for edge routes expose the deployment and its HTTP port (e.g. 8080):
    ```
    $ oc expose deployment/go-troubleshroute --port=8080 --name=go-troubleshroute-http
    $ oc expose svc/go-troubleshroute-http
    ```

* for re-rencrypt routes see `make ocp-deploy` for a simple setup

* using a backend serving specific SNI certificates and a fallback/catchall certificate:
    ```
    $ CERT_CN=*.apps.example.com PKI_DIR=pki/default make pki
    <...>
    $ CERT_CN=myroute.apps.example.com PKI_DIR=pki/myroute make pki
    <...>

    EXTRA_OPTS='-e TLS_CERT=/usr/src/app/pki/default/tls.crt -e TLS_KEY=/usr/src/app/pki/default/tls.key -e SERVER_TLS_CERT=/usr/src/app/pki/myroute/tls.crt -e SERVER_TLS_KEY=/usr/src/app/pki/myroute/tls.key' make run-image

    connect with sni:

    $ echo /dev/null|openssl s_client -CAfile pki/myroute/cacert.pem -servername myroute.apps.example.com -connect 127.0.0.1:8443|openssl x509 -noout -subject
    depth=1 O = test, OU = Org, CN = RootCA
    verify return:1
    depth=0 O = test, OU = Org, CN = myroute.apps.example.com
    verify return:1
    DONE
    subject=O = test, OU = Org, CN = myroute.apps.example.com


   connecti w/o sni:

   $ echo /dev/null|openssl s_client -CAfile pki/default/cacert.pem -connect 127.0.0.1:8443|openssl x509 -noout -subject
   Can't use SSL_get_servername
   depth=1 O = test, OU = Org, CN = RootCA
   verify return:1
   depth=0 O = test, OU = Org, CN = *.apps.example.com
   verify return:1
   DONE
   subject=O = test, OU = Org, CN = *.apps.example.com

   ```

Available endpoints
-------------------

| endpoint | response |
| --- | --- |
|/ | Basic connection info       |
|/headers | dump request headers       |
|/healthz | health check       |


