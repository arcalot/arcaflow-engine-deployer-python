FROM quay.io/centos/centos:stream8@sha256:86ba0bf60249e6daee350459e014213742c88341e6e7284695dcf1dfe2c58873

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]