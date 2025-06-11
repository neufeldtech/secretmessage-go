FROM mcr.microsoft.com/devcontainers/go:1.23

# [Optional] Uncomment this section to install additional OS packages.
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends jq

# [Optional] Uncomment the next lines to use go get to install anything else you need
USER vscode
# RUN go get -x <your-dependency-or-tool>
# USER root

# [Optional] Uncomment this line to install global node packages.
# RUN su vscode -c "source /usr/local/share/nvm/nvm.sh && npm install -g <your-package-here>" 2>&1
COPY --from=hairyhenderson/gomplate:stable /gomplate /bin/gomplate

RUN go install github.com/air-verse/air@v1.61.7
