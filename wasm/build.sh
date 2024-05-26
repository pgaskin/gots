#!/bin/sh

set -e
cd "$(set -e; dirname "$0")"
cd ..

trap 'rm -f *.o *.tmp a.out' EXIT

test "$1" = "build" || exec docker run --rm -v "$(pwd):/src" -w /src -u $(id -u):$(id -g) --entrypoint wasm/build.sh ghcr.io/webassembly/wasi-sdk:wasi-sdk-22@sha256:a508461a49ebde247a83ae605544896a3ef78d983d7a99544b8dc3c04ff2b211 build
set -u
set -x

CFLAGS="$CFLAGS -Wall -Oz"
CXXFLAGS="$CXXFLAGS -Wall -Oz -fno-exceptions"

$CC $CFLAGS -c -std=c11 -I./lz4/lib \
 ./lz4/lib/lz4.c \
 ./lz4/lib/lz4file.c \
 ./lz4/lib/lz4frame.c \
 ./lz4/lib/lz4hc.c \
 ./lz4/lib/xxhash.c

$CC $CFLAGS -c -std=c11 -I./libz -DZ_INSIDE_LIBZ -D_GNU_SOURCE -Wno-incompatible-pointer-types-discards-qualifiers \
 ./libz/adler32.c \
 ./libz/compress.c \
 ./libz/crc32.c \
 ./libz/deflate.c \
 ./libz/gzclose.c \
 ./libz/gzlib.c \
 ./libz/gzread.c \
 ./libz/gzwrite.c \
 ./libz/infback.c \
 ./libz/inffast.c \
 ./libz/inflate.c \
 ./libz/inftrees.c \
 ./libz/trees.c \
 ./libz/uncompr.c \
 ./libz/zutil.c

$CC $CFLAGS -c -std=c11 -I./brotli/c/include \
 ./brotli/c/common/constants.c \
 ./brotli/c/common/context.c \
 ./brotli/c/common/dictionary.c \
 ./brotli/c/common/platform.c \
 ./brotli/c/common/shared_dictionary.c \
 ./brotli/c/common/transform.c \
 ./brotli/c/dec/bit_reader.c \
 ./brotli/c/dec/decode.c \
 ./brotli/c/dec/huffman.c \
 ./brotli/c/dec/state.c

$CXX $CXXFLAGS -c -std=c++11 -isystem./brotli/c/include -I./woff2/include -Wno-unused-variable -Wno-unused-const-variable \
 ./woff2/src/table_tags.cc \
 ./woff2/src/variable_length.cc \
 ./woff2/src/woff2_common.cc \
 ./woff2/src/woff2_dec.cc \
 ./woff2/src/woff2_out.cc

$CXX $CXXFLAGS -c -std=c++11 -isystem./woff2/include -isystem./libz -isystem./lz4/lib -I./ots/include -DPACKAGE=ots -DVERSION=9.1.0 -DOTS_GRAPHITE=1 -DOTS_SYNTHESIZE_MISSING_GVAR=1 -DOTS_COLR_CYCLE_CHECK=1 \
 ./ots/src/avar.cc \
 ./ots/src/cff.cc \
 ./ots/src/cff_charstring.cc \
 ./ots/src/cmap.cc \
 ./ots/src/colr.cc \
 ./ots/src/cpal.cc \
 ./ots/src/cvar.cc \
 ./ots/src/cvt.cc \
 ./ots/src/fpgm.cc \
 ./ots/src/fvar.cc \
 ./ots/src/gasp.cc \
 ./ots/src/gdef.cc \
 ./ots/src/glyf.cc \
 ./ots/src/gpos.cc \
 ./ots/src/gsub.cc \
 ./ots/src/gvar.cc \
 ./ots/src/hdmx.cc \
 ./ots/src/head.cc \
 ./ots/src/hhea.cc \
 ./ots/src/hvar.cc \
 ./ots/src/kern.cc \
 ./ots/src/layout.cc \
 ./ots/src/loca.cc \
 ./ots/src/ltsh.cc \
 ./ots/src/math.cc \
 ./ots/src/maxp.cc \
 ./ots/src/metrics.cc \
 ./ots/src/mvar.cc \
 ./ots/src/name.cc \
 ./ots/src/os2.cc \
 ./ots/src/ots.cc \
 ./ots/src/post.cc \
 ./ots/src/prep.cc \
 ./ots/src/stat.cc \
 ./ots/src/variations.cc \
 ./ots/src/vdmx.cc \
 ./ots/src/vhea.cc \
 ./ots/src/vorg.cc \
 ./ots/src/vvar.cc \
 ./ots/src/feat.cc \
 ./ots/src/glat.cc \
 ./ots/src/gloc.cc \
 ./ots/src/sile.cc \
 ./ots/src/silf.cc \
 ./ots/src/sill.cc

$CXX $CXXFLAGS -c -std=c++11 -isystem./ots/include -mmultivalue ./wasm/main.cc
$CXX $CXXFLAGS -Wl,--no-entry *.o -nostartfiles -o wasm/ots.wasm
