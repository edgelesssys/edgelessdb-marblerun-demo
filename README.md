# EdgelessDB 🤝 MarbleRun

## Requirements
1. A fairly recent build of MarbleRun (v0.4.0 is still too early)
2. EGo
3. Docker

## Environment
We currently expect that you run this locally on a dev setup.

## Howto
1. Generate signing key
```bash
openssl genrsa -out private.pem -3 3072
```

2. Build reader/writer application
```bash
cd reader
ego-go build reader.go
ego sign reader
cd ../writer
ego-go build writer.go
ego sign writer
cd ..
```

3. Retrieve UniqueID / SignerID, enter in MarbleRun manifest
```bash
ego uniqueid reader/reader
ego uniqueid writer/writer
```

4. Launch the Coordinator
```bash
erthost ~/marblerun/build/coordinator-enclave.signed
```

5. Deploy MarbleRun manifest
```bash
curl -k --data-binary @marblerun-manifest.json https://localhost:4433/manifest
```
6. Launch EdgelessDB as Marble:
```bash
docker run --network host --name my-edb --privileged -e "EDG_MARBLE_TYPE=edgelessdb_marble" -e "EDG_MARBLE_COORDINATOR_ADDR=localhost:2001" -e "EDG_MARBLE_UUID_FILE=uuid" -e "EDG_MARBLE_DNS_NAMES=localhost" -v /dev/sgx:/dev/sgx -t ghcr.io/edgelesssys/edgelessdb-sgx-4gb -marble
```

7. Run the reader:
```bash
EDG_MARBLE_TYPE=reader EDG_MARBLE_COORDINATOR_ADDR=localhost:2001 EDG_MARBLE_UUID_FILE=~/reader-uuid EDG_MARBLE_DNS_NAMES=localhost ego marblerun reader/reader
```

8. Run the writer:
```bash
EDG_MARBLE_TYPE=writer EDG_MARBLE_COORDINATOR_ADDR=localhost:2001 EDG_MARBLE_UUID_FILE=~/writer-uuid EDG_MARBLE_DNS_NAMES=localhost ego marblerun writer/writer
```

9. Visit "http://localhost:8008"

10. You should see new user data popping up every 10 seconds.

You can verify the identity of the running MarbleRun cluster via attestation:
```bash
era -c ~/marblerun/build/coordinator-config.json -h localhost:4433 -output-chain marblerun-chain.pem
```

The deployed manifest can be verified via the SGX DCAP quote which can be queried over MarbleRun's `/quote` HTTP REST API endpoint. If you can verify the MarbleRun instance and manifest, you can also automatically verify EdgelessDB and the deployed manifest. However, if you like you can also additionally attestate the running EdgelessDB instance over its `/quote` HTTP REST API endpoint and compare it with the output from MarbleRun.

## Howto on Kubernetes

1. Install MarbleRun

    Using the CLI

    ```bash
    marblerun install
    ```

    Wait for the control plane to finish installing

    ```bash
    marblerun check
    ```

1. Port forward the Coordinator's Client API

    ```bash
    kubectl -n marblerun port-forward svc/coordinator-client-api 4433:4433 --address localhost >/dev/null &
    export MARBLERUN=localhost:4433
    ```

1. Deploy the MarbleRun manifest
    
    ```bash
    marblerun manifest set marblerun-manifest.json $MARBLERUN
    ```

1. Launch EdgelessDB

    * Create and annotate the target namespace
        ```bash
        kubectl create namespace edgelessdb
        marblerun namespace add edgelessdb
        ```

    * Deploy the application using helm
        ```bash
        helm install -f ./kubernetes-edb/values.yaml edgelessdb ./kubernetes-edb -n edgelessdb --set edb.launchMarble=true
        ```

    * Post forward EDB's Client API
        ```bash
        kubectl -n edgelessdb port-forward svc/edgelessdb-rest-api 8080:8080 --address localhost >/dev/null &
        ```


1. Attest the MarbleRun cluster and retrieve the certificate chain
    
    ```bash
    marblerun certificate chain $MARBLERUN -o marblerun-chain.pem
    ```

1. Deploy the EdgelessDB manifest

    ```bash
    curl --cacert marblerun-chain.pem --data-binary @edb-manifest.json https://localhost:8080/manifest
    ```

1. Launch `writer` and `reader` applications

    * Create and annotate the demo namespace
        ```bash
        kubectl create namespace edb-demo
        marblerun namespace add edb-demo
        ```
    
    * Deploy using helm
        ```bash
        helm install -f ./kubernetes-client/values.yaml edb-demo ./kubernetes-client -n edb-demo
        ```

1. Port forward the reader's web interface

    ```bash
    kubectl -n edb-demo port-forward svc/edb-reader-http 8008:8008 --address localhost >/dev/null &
    ```

1. Vists "http://localhost:8008"

1. You should see new user data popping up every 10 seconds.

## Docker

1. Generate signing key

```bash
openssl genrsa -out private.pem -3 3072
```

2. Build the reader image

```bash
docker buildx build --secret id=signingkey,src=private.pem --target release_reader --tag ghcr.io/edgelesssys/edb-demo/reader:latest .
```

3. Build the writer image

```bash
docker buildx build --secret id=signingkey,src=private.pem --target release_writer --tag ghcr.io/edgelesssys/edb-demo/writer:latest .
```


## To-Do
* Write this more user-friendly
