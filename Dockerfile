# syntax=docker/dockerfile:experimental

FROM alpine/git:latest AS pull
#RUN git clone https://github.com/edgelesssys/edgelessdb-marblerun-demo /edb-demo
COPY . /edb-demo

FROM ghcr.io/edgelesssys/ego-dev:latest AS demo_build
COPY --from=pull /edb-demo /edb-demo

WORKDIR /edb-demo/reader
RUN ego-go build reader.go
RUN --mount=type=secret,id=signingkey,dst=/edb-demo/private.pem,required=true ego sign reader

WORKDIR /edb-demo/writer
RUN ego-go build writer.go
RUN --mount=type=secret,id=signingkey,dst=/edb-demo/private.pem,required=true ego sign writer

FROM ghcr.io/edgelesssys/ego-deploy:latest AS release_reader
LABEL descritpion="EdgelessDB demo writer"
COPY --from=demo_build /edb-demo/reader/reader /
ENTRYPOINT [ "ego", "marblerun", "/reader" ]

FROM ghcr.io/edgelesssys/ego-deploy:latest AS release_writer
LABEL descritpion="EdgelessDB demo reader"
COPY --from=demo_build /edb-demo/writer/writer /
ENTRYPOINT [ "ego", "marblerun", "/writer" ]
