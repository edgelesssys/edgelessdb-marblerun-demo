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

7. Attest the MarbleRun cluster:
```bash
era -c ~/marblerun/build/coordinator-config.json -h localhost:4433 -output-chain marblerun-chain.pem
```

8. Deploy EdgelessDB manifest:
```bash
curl --cacert marblerun-chain.pem --data-binary @edb-manifest.json https://localhost:8080/manifest
```

9. Run the reader:
```bash
EDG_MARBLE_TYPE=reader EDG_MARBLE_COORDINATOR_ADDR=localhost:2001 EDG_MARBLE_UUID_FILE=~/reader-uuid EDG_MARBLE_DNS_NAMES=localhost ego marblerun reader/reader
```

10. Run the writer:
```bash
EDG_MARBLE_TYPE=writer EDG_MARBLE_COORDINATOR_ADDR=localhost:2001 EDG_MARBLE_UUID_FILE=~/writer-uuid EDG_MARBLE_DNS_NAMES=localhost ego marblerun writer/writer
```

11. Visit "http://localhost:8008"

12. You should see new user data popping up every 10 seconds.

## To-Do
* Write this more user-friendly
* Get this running in Kubernetes 😭😭😭
