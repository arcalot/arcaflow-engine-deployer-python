FROM quay.io/centos/centos:stream8@sha256:1338a5c9e232a419978dbb2f6f42a71553e0ed53f3332b618f89a19143070d86

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]