FROM quay.io/centos/centos:stream8@sha256:35fbcb91cfbaa45eea2497abacbaf67d7df35efa8ff5ccef36e077cf5ac970f7

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]