FROM quay.io/centos/centos:stream8@sha256:02cbd1a3618827fa94d43fd35c116ce619b98cc79a1788db4913cfb74c3cc3b4

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]