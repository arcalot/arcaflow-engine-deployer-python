FROM quay.io/centos/centos:stream8@sha256:abb60170a002e1a9de6aeeb0ce9b3a8248dd202d5247621a548aab2d1c09ecd5

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]