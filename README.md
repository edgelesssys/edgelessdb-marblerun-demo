# EdgelessDB 🤝 MarbleRun

This demo showcases the integration of [EdgelessDB](https://github.com/edgelesssys/edgelessdb), a MySQL-compatible database running in an enclave, with [MarbleRun](https://github.com/edgelesssys/marblerun), the control plane for confidential computing.

We will be creating two client applications to interact with EdgelessDB: one to write information to the database; and one to read, and serve it on a web-interface.
When using EdgelessDB as a standalone application, one needs to manage certificates for connecting clients, to ensure only privileged clients are allowed to manipulate data.
Using MarbleRun, we can automate secure distribution of certificates to both clients and EdgelessDB itself, while also verifying clients run inside enclaves under predefined configurations.


## Environment
We currently expect that you run this locally on a dev setup.

## Local Deployment

Local Deployment assumes you have access to an SGX capable machine and MarbleRun is ready to go.
While not necessary, for ease of use we also assume MarbleRun, EdgelessDB and the two client applications, all run on the same machine.

### Requirements
1. [MarbleRun](https://github.com/edgelesssys/marblerun) v0.5.0 or newer
2. [EGo](https://github.com/edgelesssys/ego) v0.3.2 or newer
3. [Docker](https://www.docker.com/)

### Howto

1. Generate a signing key

    This key will be used to sign the client applications
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

    You can download the signed binary from [MarbleRun's releases page](https://github.com/edgelesssys/marblerun/releases), or you can follow the [build instructions](https://github.com/edgelesssys/marblerun/blob/master/BUILD.md) to built your own.
    ```bash
    erthost coordinator-enclave.signed
    ```

5. Deploy the MarbleRun manifest

    ```bash
    curl -k --data-binary @marblerun-manifest.json https://localhost:4433/manifest
    ```

6. Launch EdgelessDB as a Marble

    ```bash
    docker run -it --network host --name my-edb --privileged -e "EDG_MARBLE_TYPE=edgelessdbMarble" -e "EDG_MARBLE_COORDINATOR_ADDR=localhost:2001" -e "EDG_MARBLE_UUID_FILE=uuid" -e "EDG_MARBLE_DNS_NAMES=localhost" -v /dev/sgx:/dev/sgx -t ghcr.io/ edgelesssys/edgelessdb-sgx-4gb -marble
    ```

    Usually, when running EdgelessDB without MarbleRun, users are required to upload a manifest to initialize EdgelessDB. With MarbleRun however, we can let the Coordinator take care of distributing this manifest.
    Taking a look at `marblerun-manifest.json`, the Marble `edgelessdbMarble` specifies a base64 encoded file `/data/manifest.json`. The `Data` field is the base64 encoding of the file `edb-manifest.json`.
    Upon start of EdgelessDB as a Marble, this file will be provided to the Marble by the Coordinator, saving us the manual initialization.

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

        Doing this allows MarbleRun's admission controller to inject SGX resources and MarbleRun specific environment variables into the starting Marble Pod.
        This saves us having to manually specify these values and allows for device plugin independent Helm charts.
        ```bash
        kubectl create namespace edgelessdb
        marblerun namespace add edgelessdb
        ```

    * Deploy the application using Helm
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
