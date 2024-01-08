FROM quay.io/centos/centos:stream8@sha256:a8692b39e546eed9177d495db1edfd97bb6de70b9527f58aeb72f90b687c3426

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]