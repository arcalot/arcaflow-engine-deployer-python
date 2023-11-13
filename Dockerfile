FROM quay.io/centos/centos:stream8@sha256:f47f028f2ad182b6784c1fecc963cb4e5914f70e413a1a4fe852f92bf855c17d

COPY tests/test_script.sh /
RUN dnf install net-tools -y

ENTRYPOINT [ "bash", "test_script.sh" ]