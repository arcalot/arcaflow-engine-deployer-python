FROM quay.io/centos/centos:stream8@sha256:4f3138f56a39ecc9289457931cd10b5f696639c7e1b060d88865c3002169abad

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]