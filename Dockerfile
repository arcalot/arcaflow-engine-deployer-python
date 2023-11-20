FROM quay.io/centos/centos:stream8@sha256:ce6ec049788dd34c9fd99cf6c319a1cc69579977b8433d00cc982df6b75841f6

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]