# EdgelessDB 🤝 MarbleRun


## Environment
We currently expect that you run this locally on a dev setup.

## Local Deployment

### Requirements
1. [MarbleRun](https://github.com/edgelesssys/marblerun) v0.5.0 or newer
2. EGo
3. Docker

### Howto

1. Generate a signing key
    ```bash
    openssl genrsa -out private.pem -3 3072
    ```

2. Build the reader and writer Marbles
    ```bash
    cd reader
    ego-go build reader.go
    ego sign reader
    cd ../writer
    ego-go build writer.go
    ego sign writer
    cd ..
    ```

3. Retrieve SignerID of the Marbles
    ```bash
    ego signerid reader/reader
    ego signerid writer/writer
    ```

    Set the output of the previous commands in `marblerun-manifest.json` as `SignerID` for `reader` and `writer`:
    ```javascript
    "reader": {
            "Debug": true,
            "SecurityVersion": 1,
            "ProductID": 17,
            "SignerID": "<YOUR_READER_SIGNER_ID>"
        },
        "writer": {
            "Debug": true,
            "SecurityVersion": 1,
            "ProductID": 18,
            "SignerID": "<YOUR_WRITER_SIGNER_ID>"
        }
    ```

4. Launch the Coordinator
    ```bash
    erthost ~/marblerun/build/coordinator-enclave.signed
    ```

5. Deploy the MarbleRun manifest
    ```bash
    curl -k --data-binary @marblerun-manifest.json https://localhost:4433/manifest
    ```

6. Launch EdgelessDB as a Marble
    ```bash
    docker run -it --network host --name my-edb --privileged -e "EDG_MARBLE_TYPE=edgelessdb_marble" -e "EDG_MARBLE_COORDINATOR_ADDR=localhost:2001" -e "EDG_MARBLE_UUID_FILE=uuid" -e "EDG_MARBLE_DNS_NAMES=localhost" -v /dev/sgx:/dev/sgx -t ghcr.io/ edgelesssys/edgelessdb-sgx-4gb -marble
    ```

7. Run the reader Marble
    ```bash
    EDG_MARBLE_TYPE=reader EDG_MARBLE_COORDINATOR_ADDR=localhost:2001 EDG_MARBLE_UUID_FILE=~/reader-uuid EDG_MARBLE_DNS_NAMES=localhost ego marblerun reader/reader
    ```

8. Run the writer Marble
    ```bash
    EDG_MARBLE_TYPE=writer EDG_MARBLE_COORDINATOR_ADDR=localhost:2001 EDG_MARBLE_UUID_FILE=~/writer-uuid EDG_MARBLE_DNS_NAMES=localhost ego marblerun writer/writer
    ```

9. Verify the MarbleRun cluster
    Verify the identity of the running MarbleRun cluster via remote attestation using [era](https://github.com/edgelesssys/era):
    ```bash
    era -c ~/marblerun/build/coordinator-config.json -h localhost:4433 -output-chain marblerun-chain.pem
    ```

    The deployed manifest can be verified via the SGX DCAP quote which can be queried over MarbleRun's `/quote` HTTP REST API endpoint. If you can verify the MarbleRun instance and manifest, you can also automatically verify EdgelessDB and the deployed manifest. However, if you like you can also additionally attest the running EdgelessDB instance over its `/quote` HTTP REST API endpoint and compare it with the output from MarbleRun.

10. Connect to the readers web interface

    Visit "https://localhost:8008"

    You’ll be presented with a certificate warning because your browser does not know MarbleRun’s root certificate as a root of trust. You can safely ignore this message, since the authenticity of the MarbleRun instance has been verified in the previous step. Alternatively you can add the certificate generated as output from the previous step as a root of trust to your browser.

    Once connected you should see new user data popping up every 10 seconds.


## Kubernetes Deployment

### Requirements
1. A Kubernetes cluster at v1.18 or newer, with SGX device-plugin installed
1. [MarbleRun CLI](https://docs.edgeless.systems/marblerun/#/reference/cli?id=installation)
1. [Helm](https://helm.sh/docs/intro/) at v3 or newer

### Howto 

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

1. Launch `writer` and `reader` Marbles

    * Create and annotate the demo namespace
        ```bash
        kubectl create namespace edb-demo
        marblerun namespace add edb-demo
        ```
    
    * Deploy using Helm
        ```bash
        helm install -f ./kubernetes-client/values.yaml edb-demo ./kubernetes-client -n edb-demo
        ```

1. Retrieve the Coordinator's certificate chain
    ```bash
    marblerun certificate chain $MARBLERUN -o marblerun.crt
    ```

1. Connect to the readers web interface
    
    Forward the interface to localhost
    ```bash
    kubectl -n edb-demo port-forward svc/edb-reader-http 8008:8008 --address localhost >/dev/null &
    ```

    Visit "https://localhost:8008"

    You’ll be presented with a certificate warning because your browser does not know MarbleRun’s root certificate as a root of trust. You can safely ignore this message, since the authenticity of the MarbleRun instance has been verified every time the MarbleRun CLI connected to the Coordinator. Alternatively you can add the certificate generated as output from the previous step as a root of trust to your browser.

    Once connected you should see new user data popping up every 10 seconds.

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

    Note that you might need to change the tag in the Helm charts if you want to run a locally built image.

## To-Do
* Add some intro text explaining what the demo does