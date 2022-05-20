# -*- mode: makefile -*-

REGISTRY ?= quay.io
IMAGE ?= go-troubleshroute
QUAY_NAMESPACE ?= bverschueren
TAG ?= latest
LOCAL_HTTPS_PORT ?= 8443
LOCAL_HTTP_PORT ?= 8080
CERT_CN ?= go-troubleshroute.apps.test.example.com

image:
	podman build -f Dockerfile -t $(REGISTRY)/$(QUAY_NAMESPACE)/$(IMAGE):$(TAG)

push-image: image
	podman push $(REGISTRY)/$(QUAY_NAMESPACE)/$(IMAGE):$(TAG)

binary-local:
	go build -v -o ./bin/go-troubleshroute ./...

binary:
	podman run --rm --volume $(PWD):/usr/src/go-troubleshroute:Z -w /usr/src/go-troubleshroute golang:1.20 go build -v -o ./bin/go-troubleshroute

pki:
	mkdir -p ./pki
	openssl genrsa -out ./pki/ca.key 4096
	openssl req -new -x509 -days 365 -key ./pki/ca.key -out ./pki/cacert.pem -subj "/O=test/OU=Org/CN=RootCA"
	openssl genrsa -out ./pki/tls.key 4096
	openssl req -new -key ./pki/tls.key -out ./pki/server.csr -subj "/O=test/OU=Org/CN=$(CERT_CN)"
	openssl x509 -req -in ./pki/server.csr  -CA ./pki/cacert.pem -CAkey ./pki/ca.key -out ./pki/tls.crt -CAcreateserial -days 365 -sha256

run:	binary pki
	go run server.go

test:
	podman run --rm --volume $(PWD)/pki/:/usr/src/app/pki:Z --volume $(PWD):/usr/src/go-troubleshroute:Z -w /usr/src/go-troubleshroute golang:1.20 go test

run-image: image pki
	podman run --rm --volume $(PWD)/pki/:/usr/src/app/pki:Z --publish $(LOCAL_HTTPS_PORT):443 --publish $(LOCAL_HTTP_PORT):80 $(REGISTRY)/$(QUAY_NAMESPACE)/$(IMAGE):$(TAG)

clean-pki:
	rm -rf ./pki/

clean-bin:
	rm -rf ./bin/

clean:	clean-pki clean-bin

ocp-deploy: pki
	oc create secret tls go-troubleshroute-tls --key pki/tls.key --cert pki/tls.crt
	oc create deployment go-troubleshroute --image=$(REGISTRY)/$(QUAY_NAMESPACE)/$(IMAGE):$(TAG)
	oc set env deployment/go-troubleshroute HTTPS_PORT=8443 HTTP_PORT=8080
	oc expose deployment/go-troubleshroute --target-port 8443 --port 443
	oc create route reencrypt --dest-ca-cert=pki/cacert.pem --service=go-troubleshroute
