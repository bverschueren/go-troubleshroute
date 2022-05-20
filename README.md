go-troubleshroute
========

A simple Go-based http(s) responder for testing OCP route termination configurations.

Deployment
----------

Run a local instance:

    $ make run
    
or in a container:

    $ make run-container

or for more flexibility:

    $ podman run --rm --volume $(pwd)/pki/:/usr/src/app/pki:Z --publish <LOCAL_PORT>:443 quay.io/bverschueren/go-troubleshroute:latest


Run as a deployment:

    $ CERT_CN=myapp.apps.mycluster.example.com make pki
    $ make ocp-deploy

Configuration
-------------

By default, the "server" listens on `*:${HTTP_PORT}` + `*:${HTTPS_PORT}` offering `${TLS_CERT}` protected by key file `${TLS_CERT}` with defaults:


| ENV | Default |
| --- | --- |
|HTTPS_PORT | 443       |
|HTTP_PORT | 80       |
|TLS_CERT    | /usr/src/app/pki/tls.crt |
|TLS_KEY     | /usr/src/app/pki//tls.key |

## How is this useful?

Some examples to troubelshoot/verify routes:

* for edge routes expose the deployment and its HTTP port (e.g. 8080):
    ```
    $ oc expose deployment/go-troubleshroute --port=8080 --name=go-troubleshroute-http
    $ oc expose svc/go-troubleshroute-http
    ```

* for re-rencrypt routes see `make ocp-deploy` for a simple setup

Available endpoints
-------------------

| endpoint | response |
| --- | --- |
|/ | Basic connection info       |
|/headers | dump request headers       |
|/healthz | health check       |


