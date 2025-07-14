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

# To build for spark-bare-expo-react-native-app
bare-make generate --platform ios --arch arm64 --simulator && bare-make build && bare-make install
# This seems to be necessary to build/install an additional target, otherwise it reuses the previous target:
rm -rf build

bare-make generate --platform ios --arch arm64 && bare-make build && bare-make install
rm -rf build

bare-make generate --platform ios --arch x64 && bare-make build && bare-make install
rm -rf build

# Target the current platform
bare-make generate && bare-make build && bare-make install

bare index.js
```
