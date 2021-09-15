# EdgelessDB 🤝 MarbleRun

This demo showcases the integration of [EdgelessDB](https://github.com/edgelesssys/edgelessdb), a MySQL-compatible database running in an enclave, with [MarbleRun](https://github.com/edgelesssys/marblerun), the control plane for confidential computing.

The setup should resemble a typical microservice architecture. EdgelessDB provides the Confidential Computing-grade database and MarbleRun orchestrates the whole deployment.

We will be creating two client applications to interact with EdgelessDB: one to write information to the database; one to read, and serve it on a web interface.
When using EdgelessDB as a standalone application, one needs to manage certificates for connecting clients, to ensure only privileged clients are allowed to modify or access data.
Using MarbleRun, we can automate the secure distribution of certificates to both clients and EdgelessDB itself.
In addition, Marblerun also verifies all clients that try to access the database run inside enclaves with predefined configurations.

For more information please refer to our [documentation](https://www.edgeless.systems/resources/docs/), or feel free to open an issue in one of our GitHub repositories.

## Local Deployment

Local deployment assumes you have access to an SGX-capable machine and MarbleRun is ready to go.
While not necessary, for ease of use we also assume MarbleRun, EdgelessDB, and the two client applications, all run on the same machine.

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

    You can download the signed binary from [MarbleRun's releases page](https://github.com/edgelesssys/marblerun/releases), or you can follow the [build instructions](https://github.com/edgelesssys/marblerun/blob/master/BUILD.md) to build your own.

    ```bash
    erthost coordinator-enclave.signed
    ```

5. Deploy the MarbleRun manifest

    ```bash
    curl -k --data-binary @marblerun-manifest.json https://localhost:4433/manifest
    ```

    * Breakdown of the manifest:

        MarbleRun's manifest declares the key properties of your cluster. Your manifest needs to declare at least `Packages` and `Marbles`, for this demo we also make use of `Secrets`.

        * [`Packages`](https://docs.edgeless.systems/marblerun/#/workflows/define-manifest?id=packages): This section lists the secure enclave software packages your application uses. Here we define the properties of our used writer and reader binaries, as well as the properties of EdgelessDB, which can be found on the [releases page](https://github.com/edgelesssys/edgelessdb/releases) on GitHub.
        * [`Marbles`](https://docs.edgeless.systems/marblerun/#/workflows/define-manifest?id=marbles): Marbles represent the actual service in your mesh. They define the runtime environment of a `Package`. This includes files, environment variables, and command-line arguments.
        For reader and writer, we set two environment variables to hold a client certificate and the matching private key. These will be used by the applications to authenticate themselves to EdgelessDB. The certificates are signed by MarbleRun's Coordinator. For the reader, we additionally set the env variable `PORT` at which the webpage can be reached.
        For EdgelessDB we add four environment variables, and a file, necessary to launch the database as a Marble. For more information take a look at our [documentation](https://docs.edgeless.systems/edgelessdb/#/advanced/marblerun?id=extend-the-marblerun-manifest).
        * [`Secrets`](https://docs.edgeless.systems/marblerun/#/workflows/define-manifest?id=secrets): The `Secrets` section allows us to define secrets referenced in the `Marbles` section. Parameters include the type, key length, and for certificates, X.509 certificate properties. For the demo, we use three Elliptic Curve certificates: two for the reader and writer applications, and one for EdgelessDB. We also use one 128 Bit symmetric key as EdgelessDB's masterkey.

6. Launch EdgelessDB as a Marble

    ```bash
    docker run -it --network host --name my-edb --privileged -e "EDG_MARBLE_TYPE=EdgelessDB" -e "EDG_MARBLE_COORDINATOR_ADDR=localhost:2001" -e "EDG_MARBLE_UUID_FILE=uuid" -e "EDG_MARBLE_DNS_NAMES=localhost" -v /dev/sgx:/dev/sgx -t ghcr.io/edgelesssys/edgelessdb-sgx-4gb -marble
    ```

    Usually, when running EdgelessDB without MarbleRun, users are required to upload a manifest to initialize EdgelessDB. With MarbleRun however, we let the Coordinator take care of distributing this manifest.
    Taking a look at `marblerun-manifest.json`, the Marble `EdgelessDB` specifies a base64 encoded file `/data/manifest.json`. The `Data` field is the base64 encoding of the file `edb-manifest.json`.
    Upon the initial start of EdgelessDB as a Marble, this file will be provided by the Coordinator, saving us the manual initialization.

7. Run the client applications

    We need to set a couple of environment variables to provide the applications with the correct setup to connect to MarbleRun. \
    `EDG_MARBLE_TYPE`: This associates the application with a `Marble` in the manifest \
    `EDG_MARBLE_COORDINATOR_ADDR`: The address MarbleRun's gRPC port can be reached at \
    `EDG_MARBLE_UUID_FILE`: A file where the applications UUID will be stored \
    `EDG_MARBLE_DNS_NAMES`: The domain name the Marble's certificate should be signed for

    * Run the reader Marble

        ```bash
        EDG_MARBLE_TYPE=reader EDG_MARBLE_COORDINATOR_ADDR=localhost:2001 EDG_MARBLE_UUID_FILE=~/reader-uuid EDG_MARBLE_DNS_NAMES=localhost ego marblerun reader/reader
        ```

    * Run the writer Marble

        ```bash
        EDG_MARBLE_TYPE=writer EDG_MARBLE_COORDINATOR_ADDR=localhost:2001 EDG_MARBLE_UUID_FILE=~/writer-uuid EDG_MARBLE_DNS_NAMES=localhost ego marblerun writer/writer
        ```

8. Verify the MarbleRun cluster

    Verify the identity of the running MarbleRun cluster via remote attestation using [era](https://github.com/edgelesssys/era):

    ```bash
    era -c ~/marblerun/build/coordinator-config.json -h localhost:4433 -output-chain marblerun-chain.pem
    ```

    The deployed manifest can be verified via the SGX DCAP quote which can be queried over MarbleRun's `/quote` HTTP REST API endpoint. If you can verify the MarbleRun instance and manifest, you can also automatically verify EdgelessDB and the deployed manifest. However, if you like you can also additionally attest the running EdgelessDB instance over its `/quote` HTTP REST API endpoint and compare it with the output from MarbleRun.

9. Connect to the reader's web interface

    Visit "https://localhost:8008"

    You’ll be presented with a certificate warning because your browser does not know MarbleRun’s root certificate as a root of trust. You can safely ignore this message.

    Once connected you should see new user data popping up every 10 seconds.

## Kubernetes Deployment

In the following, we will showcase how the previous workflow can easily be adapted for Kubernetes.
Making use of MarbleRun's CLI and Kubernetes integration, many tasks, such as setting the correct environment variables and SGX resources, can be automated.

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
        This saves us having to manually specify these values and allows for device plugin-independent Helm charts.

        ```bash
        kubectl create namespace edgelessdb
        marblerun namespace add edgelessdb
        ```

    * Deploy the application using Helm

        ```bash
        # Add the helm repo
        helm repo add edgeless https://helm.edgeless.systems/stable
        helm repo update
        # Install the chart
        helm install edgelessdb edgeless/edgelessdb --set edb.launchMarble=true --namespace edgelessdb
        ```

1. Launch `writer` and `reader` Marbles

    * Create and annotate the demo namespace

        ```bash
        kubectl create namespace edb-demo
        marblerun namespace add edb-demo
        ```

    * Deploy using Helm

        ```bash
        helm install -f ./kubernetes/values.yaml edb-demo ./kubernetes-client -n edb-demo
        ```

1. Connect to the reader's web interface

    Forward the interface to localhost

    ```bash
    kubectl -n edb-demo port-forward svc/edb-reader-http 8008:8008 --address localhost >/dev/null &
    ```

    Visit "https://localhost:8008"

    You’ll be presented with a certificate warning because your browser does not know MarbleRun’s root certificate as a root of trust. You can safely ignore this message.

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
