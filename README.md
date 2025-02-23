![Version](https://img.shields.io/badge/version-0.0.0-orange.svg)
[![Documentation](https://godoc.org/github.com/vigo/cvepreserve?status.svg)](https://pkg.go.dev/github.com/vigo/cvepreserve)
[![Run golangci-lint](https://github.com/vigo/cvepreserve/actions/workflows/go-lint.yml/badge.svg)](https://github.com/vigo/cvepreserve/actions/workflows/go-lint.yml)

# CVE Preserve

Few days ago, (*Feb 22, 2025*) [Mehmet][01] [asked][02] for contribution to his
project. I implemented a proof-of-concept version, and it seems to be working.

Parent project can be found:

https://github.com/mdisec/cve-url-crawling-dataset

## Installation

If you have go installation (*1.24*) on your machine:

```bash
go install github.com/vigo/cvepreserve@latest
```

then run `cvepreserve -h` for help or build from source:

```bash
git clone git@github.com:vigo/cvepreserve.git
cd cvepreserve/
go build .
```

then run `./cvepreserve -h` for help.

## Usage

Download dataset file to you local, `~100MB` json file.

```bash
curl -L -o dataset.json https://raw.githubusercontent.com/mdisec/cve-url-crawling-dataset/main/dataset.json
```

Run the executable:

```bash
./cvepreserve          # if you are building from source, auto reads from dataset.json

# or
cvepreserve -dataset "/path/to/dataset.json"
```

The result database will be saved in the directory where you run the executable.
Sqlite database name is `result.sqlite3`

---

## Contribute

Feel free to fix bugs, improve, add features! All PR’s are welcome!

1. `fork` (https://github.com/vigo/cvepreserve/fork)
1. Create your `branch` (`git checkout -b my-feature`)
1. `commit` yours (`git commit -am 'add some functionality'`)
1. `push` your `branch` (`git push origin my-feature`)
1. Than create a new **Pull Request**!

---

## License

This project is licensed under MIT (MIT)

---

This project is intended to be a safe, welcoming space for collaboration, and
contributors are expected to adhere to the [code of conduct][coc].

[01]: https://github.com/mdisec/
[02]: https://x.com/mdisec

[coc]: https://github.com/vigo/cvepreserve/blob/main/CODE_OF_CONDUCT.md
