# syntax=docker/dockerfile:1

FROM debian

EXPOSE 443 80

VOLUME /config

WORKDIR /app

ENV PATH=$PATH:/usr/local/go/bin

RUN apt-get update && apt-get install -y ca-certificates openssl && \
    apt-get install -y wget curl && \
    apt-get install -y nodejs && \
    apt-get install -y wget curl && \
    wget https://golang.org/dl/go1.20.2.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.20.2.linux-amd64.tar.gz && \
    rm go1.20.2.linux-amd64.tar.gz && \
    curl -fsSL https://deb.nodesource.com/setup_18.x | bash - && \
    apt-get install -y nodejs && \
    apt-get remove -y wget curl && \
    apt-get autoremove -y

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY package.json ./
COPY package-lock.json ./
RUN npm install

COPY . .
RUN npm run client-build && \
    chmod +x build.sh && \
    ./build.sh && \
    rm -rf /usr/local/go \
           /tmp/* \
           /var/lib/apt/lists/* \
           /var/tmp/*

WORKDIR /app/build

CMD ["./cosmos"]
