# Stage: builder
FROM karalabe/xgo-latest

RUN $ANDROID_NDK_ROOT/build/tools/make-standalone-toolchain.sh --ndk-dir=$ANDROID_NDK_ROOT --install-dir=/usr/$ANDROID_CHAIN_ARM64 --toolchain=$ANDROID_CHAIN_ARM64 --arch=arm64
WORKDIR /go/src/github.com/CovenantSQL/CovenantSQL
COPY . .
RUN make clean
RUN GOOS=android GOARCH=arm64 CC=aarch64-linux-android-gcc make release
RUN tar cvfz /CovenantSQL.tar.gz bin/cql*