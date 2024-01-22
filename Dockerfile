FROM quay.io/centos/centos:stream8@sha256:4771b563d423363f2dc28ab7ebefac1962962daf3f49d10a7fcbfc4298222866

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]