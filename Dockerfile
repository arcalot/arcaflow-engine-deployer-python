FROM quay.io/centos/centos:stream8@sha256:cc0d7d589651639d7e890b440cb2e2c63c257693c96e1f92c6f097d5a3dd9b10

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]