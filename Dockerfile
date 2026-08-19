# The container builds the site. It does not serve it.
#
# That is `decisions/0002-what-docker-is-for.md`, and it is restated here rather
# than only there because the change that would break it is a serving stage added
# to this file by somebody who never opened that record. There is no port, no
# server and no entrypoint that waits for a request, and adding one is an argument
# to have in the record rather than a line to add here.
#
# It runs the same build verb a contributor runs. A container that reimplemented
# the build would be a second procedure that drifts from the first, and the
# reproducibility check is what proves it has not: the same source built inside
# and outside has to produce identical bytes.
FROM golang:1.26.5-bookworm@sha256:53eeac89074db483fdf0ab3be1df32bf6e47562263d2d0d6baa7f26acb4957dd

# The toolchain is the one in the image, which is the one go.mod pins. `local`
# refuses to fetch another, so a mismatch is a failure here rather than a quiet
# download, and the build needs no network at all: the module graph is empty and
# the proxy is off.
ENV GOTOOLCHAIN=local \
    GOPROXY=off \
    GOFLAGS=-mod=readonly \
    CGO_ENABLED=0 \
    HOME=/home/site

# A named user rather than root. Nothing here needs to write outside its own
# directories, and a build that runs as root writes files a person then cannot
# delete without becoming root themselves.
#
# The working directory is created and handed over here rather than by WORKDIR
# below, because WORKDIR creates it as root and the verb writes its output
# directory inside it.
RUN useradd --uid 10001 --create-home --home-dir /home/site --shell /usr/sbin/nologin site \
    && mkdir -p /src /out \
    && chown 10001:10001 /src /out

WORKDIR /src
COPY --chown=10001:10001 . .

USER 10001:10001

# Compile once while the image is being made, so the run is the build rather than
# a compile followed by a build. It also fails the image here, rather than on
# somebody's first run, if the source does not compile against the pinned
# toolchain.
RUN go build ./...

# The output directory is mounted rather than baked in. An image carrying the
# rendered site is a second copy of it that goes stale, and nothing here publishes
# an image for anybody to pull.
VOLUME ["/out"]

# The verb writes into the tree's own dist/, and the result is copied to the
# mounted directory afterwards. The copy is not part of the build: it exists
# because the verb renders relative to the tree it is given, and a bind mount over
# that path cannot be cleared the way the verb clears its output directory. The
# copy carries the bytes and not the times: a mounted directory does not always
# let its own timestamps be set, and nothing downstream reads them.
ENTRYPOINT ["/bin/sh", "-c", "go run . build && cp -R dist/. /out/"]
