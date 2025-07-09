Ensure you have bare and bare-make installed:

```sh
npm i -g bare bare-make
```

On MacOS be sure to prioritize the system toolchain instead of homebrew, otherwise you'll encounter errors for bare-make commands:

```sh
export PATH="/usr/bin:$PATH"
```

As mentioned in the [simple bare addon guide](https://github.com/holepunchto/bare-snippets/tree/main/addon-support) run the following:

```sh
yarn

cd spark-frost-bare-addon

bare-make generate

# aarch64-apple-darwin or x86_64-apple-darwin depending on your system
cargo build --release --target aarch64-apple-darwin

bare-make build

bare-make install

bare index.js
```
