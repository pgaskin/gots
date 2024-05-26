# gots

Go bindings for the [OTS](https://github.com/khaledhosny/ots) font converter/sanitizer (the same one used in Firefox and Chromium) using [wazero](https://github.com/tetratelabs/wazero).

It takes TTF/TTC/OTF/WOFF/WOFF2 fonts and outputs sanitized TTF/TTC/OTF font files.

The WebAssembly [blob](./wasm/ots.wasm) can be reproduced using `go generate .`, and is licensed under the licenses for [brotli](./brotli/LICENSE), [sortix libz](./libz/zlib.h), [lz4](./lz4/LICENSE), [ots](./ots/LICENSE), and [woff2](./woff2/LICENSE).
