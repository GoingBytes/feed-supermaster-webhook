Changes:

- Remove unnecessary stuff (see the mother repo - https://github.com/umputun/feed-master - for that stuff)
- Send new news to a webhook

Build in DEV:

    just build

Build in RELEASE:

    just release
    just release_musl

Dev:

    golangci-lint run -c .golangci.yml ./...
    betteralign -apply ./...
    nilaway ./...
    deadcode ./...

    gofumpt -l -w .

Run test:

    ./feed-master-webhook --db=./feed-master.bdb --conf=./feeds.yml
