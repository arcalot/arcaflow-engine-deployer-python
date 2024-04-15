FROM quay.io/centos/centos:stream8@sha256:cb10b58aa113732193dddbbe4e91d80ca97f8ec742ef66cf4ff8340f5b90faab

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]