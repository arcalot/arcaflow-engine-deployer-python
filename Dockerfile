FROM quay.io/centos/centos:stream8@sha256:5917fa6bdbced823c488264ba03f1cfab852c15b5e47714fc8c9a074adc7cfdd

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]