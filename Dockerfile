FROM quay.io/centos/centos:stream8@sha256:34aaf8788a2467f602c5772884448236bb41dfe1691a78dee33053bb24474395

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]