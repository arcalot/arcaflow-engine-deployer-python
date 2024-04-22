FROM quay.io/centos/centos:stream8@sha256:f61b2ab26101acac38e744a81fd42d9ec4cb89cb97dc886d684bcb25bdbae3bd

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]