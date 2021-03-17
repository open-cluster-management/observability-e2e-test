FROM registry.fedoraproject.org/fedora-minimal:32 as build

ENV GOPATH /go
ENV CGO_ENABLED=0

RUN microdnf -y install --nodocs wget unzip tar git gcc

# install go into build image
RUN wget --no-check-certificate -O - 'https://dl.google.com/go/go1.14.2.linux-amd64.tar.gz' | tar xz -C /usr/local/
RUN mkdir -p /go/bin

ENV PATH usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:/go/bin:.

# copy go tests into build image
RUN mkdir -p /go/src/github.com/open-cluster-management/observability-e2e-test/
COPY . /go/src/github.com/open-cluster-management/observability-e2e-test/

WORKDIR "/go/src/github.com/open-cluster-management/observability-e2e-test/"

# compile go tests in build image
RUN go get github.com/onsi/ginkgo/ginkgo@v1.14.2 && ginkgo build

# create new docker image to hold built artifacts
FROM registry.fedoraproject.org/fedora-minimal:32

# run as root
USER root

# expose env vars for runtime
ENV KUBECONFIG "/opt/.kube/config"
ENV OPTIONS "/resources/options.yaml"
ENV REPORT_FILE "/results/results.xml"
ENV GINKGO_DEFAULT_FLAGS "-slowSpecThreshold=120 -timeout 7200s"
ENV GINKGO_NODES "1"
ENV GINKGO_FLAGS=""
ENV GINKGO_FOCUS=""
ENV GINKGO_SKIP=""

# install ginkgo into built image
COPY --from=build /go/bin/ /usr/local/bin

# copy compiled tests into built image
RUN mkdir -p /opt/tests
COPY --from=build /go/src/github.com/open-cluster-management/observability-e2e-test/observability-e2e-test.test /opt/tests

VOLUME /results
WORKDIR "/opt/tests/"

# execute compiled ginkgo tests
CMD ["/bin/bash", "-c", "ginkgo --v --focus=${GINKGO_FOCUS} --skip=${GINKGO_SKIP} -nodes=${GINKGO_NODES} --reportFile=${REPORT_FILE} -x -debug -trace observability-e2e-test.test -- -v=3"]
