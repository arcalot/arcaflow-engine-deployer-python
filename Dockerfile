FROM quay.io/centos/centos:stream8@sha256:e7f228fe74eeac927a3133ae78a75aac1f28f6dff284616a7b2b10b5769b6275

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]