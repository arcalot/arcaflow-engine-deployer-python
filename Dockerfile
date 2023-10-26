FROM quay.io/centos/centos:stream8@sha256:b1f6889548eda34b2ddc8c2f50a49bf9924164814308e41e90a07e3b30e0db7f

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]