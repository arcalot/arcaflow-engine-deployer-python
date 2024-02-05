FROM quay.io/centos/centos:stream8@sha256:20ef8d90e1bd590f614dccb6e24d612bd4fc85fbe394f2395f103b6aa7140c4d

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]